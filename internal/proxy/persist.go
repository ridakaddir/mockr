package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
)

// applyPersist reads the stub file, mutates the array according to merge strategy,
// writes the file back, and sends the appropriate HTTP response.
//
// Returns true if it handled the response (caller should not write anything else).
func applyPersist(w http.ResponseWriter, r *http.Request, c config.Case, bodyBytes []byte, routePattern string, configDir string) bool {
	if !c.Persist {
		return false
	}

	filePath := c.File
	if hasDynamicPlaceholders(filePath) {
		filePath = resolveDynamicFile(filePath, r, bodyBytes)
	}
	if configDir != "" && !filepath.IsAbs(filePath) {
		filePath = filepath.Join(configDir, filePath)
	}

	// Read existing stub file.
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("persist: reading stub file", "file", filePath, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file read error"})
		return true
	}

	var root map[string]interface{}
	if err := json.Unmarshal(fileData, &root); err != nil {
		logger.Error("persist: parsing stub file", "file", filePath, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file parse error"})
		return true
	}

	// Parse request body.
	var incoming map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &incoming); err != nil {
			logger.Warn("persist: could not parse request body as JSON")
		}
	}

	switch strings.ToLower(c.Merge) {
	case "append":
		if err := persistAppend(root, c.ArrayKey, incoming); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		if err := writeStubFile(filePath, root); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file write error"})
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), incoming)

	case "replace":
		keyVal := resolveKeyValue(c.Key, r, bodyBytes, routePattern)
		updated, err := persistReplace(root, c.ArrayKey, c.Key, keyVal, incoming)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "record not found",
				"key":   c.Key,
				"value": keyVal,
			})
			return true
		}
		if err := writeStubFile(filePath, root); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file write error"})
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), updated)

	case "delete":
		keyVal := resolveKeyValue(c.Key, r, bodyBytes, routePattern)
		if err := persistDelete(root, c.ArrayKey, c.Key, keyVal); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "record not found",
				"key":   c.Key,
				"value": keyVal,
			})
			return true
		}
		if err := writeStubFile(filePath, root); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file write error"})
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		w.WriteHeader(c.StatusCode())

	default:
		logger.Warn("persist: unknown merge strategy", "merge", c.Merge)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("unknown merge strategy: %s", c.Merge)})
	}

	return true
}

// persistAppend adds incoming to the array at root[arrayKey].
func persistAppend(root map[string]interface{}, arrayKey string, incoming map[string]interface{}) error {
	arr, err := getArray(root, arrayKey)
	if err != nil {
		return err
	}
	root[arrayKey] = append(arr, incoming)
	return nil
}

// persistReplace finds the record where record[key] == keyVal and replaces it with incoming.
func persistReplace(root map[string]interface{}, arrayKey, key, keyVal string, incoming map[string]interface{}) (map[string]interface{}, error) {
	arr, err := getArray(root, arrayKey)
	if err != nil {
		return nil, err
	}

	for i, item := range arr {
		record, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprintf("%v", record[key]) == keyVal {
			// Merge incoming onto existing record.
			for k, v := range incoming {
				record[k] = v
			}
			arr[i] = record
			root[arrayKey] = arr
			return record, nil
		}
	}

	return nil, fmt.Errorf("not found")
}

// persistDelete finds the record where record[key] == keyVal and removes it.
func persistDelete(root map[string]interface{}, arrayKey, key, keyVal string) error {
	arr, err := getArray(root, arrayKey)
	if err != nil {
		return err
	}

	newArr := make([]interface{}, 0, len(arr))
	found := false
	for _, item := range arr {
		record, ok := item.(map[string]interface{})
		if ok && fmt.Sprintf("%v", record[key]) == keyVal {
			found = true
			continue
		}
		newArr = append(newArr, item)
	}

	if !found {
		return fmt.Errorf("not found")
	}

	root[arrayKey] = newArr
	return nil
}

// getArray returns the []interface{} stored at root[arrayKey].
func getArray(root map[string]interface{}, arrayKey string) ([]interface{}, error) {
	if arrayKey == "" {
		return nil, fmt.Errorf("array_key is required for persist operations")
	}
	raw, exists := root[arrayKey]
	if !exists {
		return nil, fmt.Errorf("array_key %q not found in stub file", arrayKey)
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("array_key %q is not a JSON array", arrayKey)
	}
	return arr, nil
}

// resolveKeyValue finds the value for key using: path wildcard → body → query.
func resolveKeyValue(key string, r *http.Request, bodyBytes []byte, routePattern string) string {
	// 1. Try path wildcard.
	if v, ok := extractWildcardValue(routePattern, r.URL.Path); ok && v != "" {
		return v
	}

	// 2. Try request body.
	if v, found := extractBodyField(key, bodyBytes); found {
		return v
	}

	// 3. Try query param.
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}

	return ""
}

// writeStubFile marshals root back to the stub file with indentation.
func writeStubFile(path string, root map[string]interface{}) error {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
