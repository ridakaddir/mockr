// Package persist contains the transport-agnostic stub file mutation logic
// shared between the HTTP and gRPC handlers.
package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Append adds incoming to the array stored at root[arrayKey] and writes the
// file back to disk.
func Append(filePath, arrayKey string, incoming map[string]interface{}) error {
	// Validate configuration matches file type
	if err := validatePersistConfig(filePath, arrayKey); err != nil {
		return err
	}

	if arrayKey == "" {
		// Bare-array mode: file is a top-level JSON array
		arr, err := ReadArray(filePath)
		if err != nil {
			return err
		}
		arr = append(arr, incoming)
		return WriteArray(filePath, arr)
	}

	// Existing wrapped-object logic
	root, err := ReadStub(filePath)
	if err != nil {
		return err
	}
	arr, err := getArray(root, arrayKey)
	if err != nil {
		return err
	}
	root[arrayKey] = append(arr, incoming)
	return WriteStub(filePath, root)
}

// Replace finds the record where record[key] == keyVal, merges incoming onto
// it, writes the file back, and returns the updated record.
func Replace(filePath, arrayKey, key, keyVal string, incoming map[string]interface{}) (map[string]interface{}, error) {
	// Validate configuration matches file type
	if err := validatePersistConfig(filePath, arrayKey); err != nil {
		return nil, err
	}

	if arrayKey == "" {
		// Bare-array mode: operate directly on the bare array
		arr, err := ReadArray(filePath)
		if err != nil {
			return nil, err
		}

		for i, item := range arr {
			record, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if fmt.Sprintf("%v", record[key]) == keyVal {
				for k, v := range incoming {
					record[k] = v
				}
				arr[i] = record
				return record, WriteArray(filePath, arr)
			}
		}

		return nil, &NotFoundError{Key: key, Value: keyVal}
	}

	// Existing wrapped-object logic
	root, err := ReadStub(filePath)
	if err != nil {
		return nil, err
	}
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
			for k, v := range incoming {
				record[k] = v
			}
			arr[i] = record
			root[arrayKey] = arr
			return record, WriteStub(filePath, root)
		}
	}

	return nil, &NotFoundError{Key: key, Value: keyVal}
}

// Delete removes the record where record[key] == keyVal and writes the file back.
func Delete(filePath, arrayKey, key, keyVal string) error {
	// Validate configuration matches file type
	if err := validatePersistConfig(filePath, arrayKey); err != nil {
		return err
	}

	if arrayKey == "" {
		// Bare-array mode: operate directly on the bare array
		arr, err := ReadArray(filePath)
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
			return &NotFoundError{Key: key, Value: keyVal}
		}

		return WriteArray(filePath, newArr)
	}

	// Existing wrapped-object logic
	root, err := ReadStub(filePath)
	if err != nil {
		return err
	}
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
		return &NotFoundError{Key: key, Value: keyVal}
	}

	root[arrayKey] = newArr
	return WriteStub(filePath, root)
}

// ReadStub reads and parses a stub JSON file into a map.
func ReadStub(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading stub file %q: %w", filePath, err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing stub file %q: %w", filePath, err)
	}
	return root, nil
}

// WriteStub marshals root and writes it back to filePath with indentation.
func WriteStub(filePath string, root map[string]interface{}) error {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling stub: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0644)
}

// detectFileType determines if a file contains a bare array or wrapped object.
func detectFileType(filePath string) (fileType string, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading stub file %q: %w", filePath, err)
	}

	// Parse as generic interface{} to detect type
	var content interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		return "", fmt.Errorf("parsing stub file %q: %w", filePath, err)
	}

	switch content.(type) {
	case []interface{}:
		return "array", nil
	case map[string]interface{}:
		return "object", nil
	default:
		return "unknown", &ConfigError{
			Msg: fmt.Sprintf("stub file %q must contain either a JSON array or object, got %T", filePath, content),
		}
	}
}

// validatePersistConfig validates that file type matches the array_key configuration.
func validatePersistConfig(filePath, arrayKey string) error {
	fileType, err := detectFileType(filePath)
	if err != nil {
		return err
	}

	if arrayKey == "" && fileType != "array" {
		return &ConfigError{
			Msg: fmt.Sprintf("stub file %q contains a JSON object but array_key is not specified. Either provide array_key to specify which field contains the array, or convert the file to a bare JSON array", filePath),
		}
	}

	if arrayKey != "" && fileType == "array" {
		return &ConfigError{
			Msg: fmt.Sprintf("stub file %q is a bare JSON array but array_key=%q is specified. Either remove array_key or wrap the array in an object like {%q: [...]}", filePath, arrayKey, arrayKey),
		}
	}

	return nil
}

// ReadArray reads a stub file as a bare JSON array.
func ReadArray(filePath string) ([]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading stub file %q: %w", filePath, err)
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		// Try to provide helpful error if it's an object
		var obj map[string]interface{}
		if json.Unmarshal(data, &obj) == nil {
			return nil, &ConfigError{
				Msg: fmt.Sprintf("stub file %q contains a JSON object, but bare-array mode expects a JSON array. Either specify array_key in your config, or convert the file to format: [{...}, {...}]", filePath),
			}
		}
		return nil, fmt.Errorf("parsing stub file %q as array: %w", filePath, err)
	}

	return arr, nil
}

// WriteArray writes a bare array back to file with proper formatting.
func WriteArray(filePath string, arr []interface{}) error {
	data, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling stub array: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0644)
}

// getArray returns the []interface{} at root[arrayKey].
func getArray(root map[string]interface{}, arrayKey string) ([]interface{}, error) {
	if arrayKey == "" {
		return nil, &ConfigError{
			Msg: "array_key is required when stub file is a JSON object. For bare JSON arrays, omit array_key entirely",
		}
	}

	raw, exists := root[arrayKey]
	if !exists {
		keys := make([]string, 0, len(root))
		for k := range root {
			keys = append(keys, fmt.Sprintf("%q", k))
		}
		return nil, &ConfigError{
			Msg: fmt.Sprintf("array_key %q not found in stub file. Available keys: [%s]", arrayKey, joinKeys(keys)),
		}
	}

	arr, ok := raw.([]interface{})
	if !ok {
		return nil, &ConfigError{
			Msg: fmt.Sprintf("field %q is not a JSON array (found %T). Ensure the field contains an array like: %q: [{...}, {...}]", arrayKey, raw, arrayKey),
		}
	}

	return arr, nil
}

// joinKeys joins a slice of strings with commas for error messages.
func joinKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	if len(keys) == 1 {
		return keys[0]
	}

	result := ""
	for i, key := range keys {
		if i > 0 {
			result += ", "
		}
		result += key
	}
	return result
}

// NotFoundError is returned when replace/delete cannot locate the target record.
type NotFoundError struct {
	Key   string
	Value string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("record not found: %s=%s", e.Key, e.Value)
}

// IsNotFound reports whether err (or any error in its chain) is a *NotFoundError.
func IsNotFound(err error) bool {
	var t *NotFoundError
	return errors.As(err, &t)
}

// ConfigError is returned for invalid persist configuration (e.g. missing or
// invalid array_key). Maps to HTTP 400 / gRPC INVALID_ARGUMENT.
type ConfigError struct {
	Msg string
}

func (e *ConfigError) Error() string { return e.Msg }

// IsConfigError reports whether err (or any error in its chain) is a *ConfigError.
func IsConfigError(err error) bool {
	var t *ConfigError
	return errors.As(err, &t)
}
