// Package persist contains the transport-agnostic stub file mutation logic
// shared between the HTTP and gRPC handlers.
package persist

import (
	"encoding/json"
	"fmt"
	"os"
)

// Append adds incoming to the array stored at root[arrayKey] and writes the
// file back to disk.
func Append(filePath, arrayKey string, incoming map[string]interface{}) error {
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

// getArray returns the []interface{} at root[arrayKey].
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

// NotFoundError is returned when replace/delete cannot locate the target record.
type NotFoundError struct {
	Key   string
	Value string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("record not found: %s=%s", e.Key, e.Value)
}

// IsNotFound reports whether err is a *NotFoundError.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
