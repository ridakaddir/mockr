package proxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/persist"
)

// applyPersist reads the stub file, mutates the array according to merge strategy,
// writes the file back, and sends the appropriate HTTP response.
//
// Returns true if it handled the response (caller should not write anything else).
func applyPersist(w http.ResponseWriter, r *http.Request, c config.Case, bodyBytes []byte, routePattern string, configDir string, pathParams map[string]string) bool {
	if !c.Persist {
		return false
	}

	filePath := resolveFilePath(c.File, r, bodyBytes, configDir, routePattern, pathParams)

	// Parse request body for incoming record.
	incoming := parseJSONBody(bodyBytes)

	switch strings.ToLower(c.Merge) {
	case "append":
		if err := persist.Append(filePath, c.ArrayKey, incoming); err != nil {
			logger.Error("persist append", "file", filePath, "err", err)
			if persist.IsConfigError(err) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file error"})
			}
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), incoming)

	case "replace":
		keyVal := resolveKeyValue(c.Key, r, bodyBytes, routePattern, pathParams)
		updated, err := persist.Replace(filePath, c.ArrayKey, c.Key, keyVal, incoming)
		if err != nil {
			if persist.IsNotFound(err) {
				writeJSON(w, http.StatusNotFound, map[string]string{
					"error": "record not found",
					"key":   c.Key,
					"value": keyVal,
				})
			} else {
				logger.Error("persist replace", "file", filePath, "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), updated)

	case "delete":
		keyVal := resolveKeyValue(c.Key, r, bodyBytes, routePattern, pathParams)
		if err := persist.Delete(filePath, c.ArrayKey, c.Key, keyVal); err != nil {
			if persist.IsNotFound(err) {
				writeJSON(w, http.StatusNotFound, map[string]string{
					"error": "record not found",
					"key":   c.Key,
					"value": keyVal,
				})
			} else {
				logger.Error("persist delete", "file", filePath, "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
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

// resolveFilePath applies dynamic placeholders and configDir to a file path.
func resolveFilePath(filePath string, r *http.Request, bodyBytes []byte, configDir string, routePattern string, pathParams map[string]string) string {
	if hasDynamicPlaceholders(filePath) {
		filePath = resolveDynamicFile(filePath, r, bodyBytes, routePattern, pathParams)
	}
	return absPath(filePath, configDir)
}

// resolveKeyValue finds the value for key using: named path params → path wildcard → body → query.
func resolveKeyValue(key string, r *http.Request, bodyBytes []byte, routePattern string, pathParams map[string]string) string {
	// 1. Try named path parameters first.
	if pathParams != nil {
		if v, ok := pathParams[key]; ok && v != "" {
			return v
		}
	}

	// 2. Try path wildcard (existing behavior for backward compatibility).
	if v, ok := extractWildcardValue(routePattern, r.URL.Path); ok && v != "" {
		return v
	}

	// 3. Try request body.
	if v, found := extractBodyField(key, bodyBytes); found {
		return v
	}

	// 4. Try query param.
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}

	return ""
}
