package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createTempArrayFile(t *testing.T, content string) string {
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	return tmpFile
}

func createTempObjectFile(t *testing.T, content string) string {
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	return tmpFile
}

func readFileAsArray(t *testing.T, filePath string) []interface{} {
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var arr []interface{}
	require.NoError(t, json.Unmarshal(data, &arr))
	return arr
}

func readFileAsObject(t *testing.T, filePath string) map[string]interface{} {
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var obj map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &obj))
	return obj
}

// === Core Bare Array Functionality Tests ===

func TestAppendBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1","name":"Alice"}]`)

	incoming := map[string]interface{}{
		"id":   "2",
		"name": "Bob",
	}

	err := Append(tmpFile, "", incoming) // empty arrayKey
	require.NoError(t, err)

	// Verify file content
	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 2)

	// Check original record preserved
	alice := result[0].(map[string]interface{})
	assert.Equal(t, "1", alice["id"])
	assert.Equal(t, "Alice", alice["name"])

	// Check new record was appended
	bob := result[1].(map[string]interface{})
	assert.Equal(t, "2", bob["id"])
	assert.Equal(t, "Bob", bob["name"])
}

func TestReplaceBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]`)

	incoming := map[string]interface{}{
		"name": "Alice Smith",
		"age":  30,
	}

	updated, err := Replace(tmpFile, "", "id", "1", incoming)
	require.NoError(t, err)

	// Check returned record
	assert.Equal(t, "Alice Smith", updated["name"])
	assert.Equal(t, 30, updated["age"]) // Returned record preserves original types
	assert.Equal(t, "1", updated["id"]) // original field preserved

	// Verify persistence
	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 2)

	alice := result[0].(map[string]interface{})
	assert.Equal(t, "Alice Smith", alice["name"])
	assert.Equal(t, float64(30), alice["age"]) // JSON numbers are float64
	assert.Equal(t, "1", alice["id"])

	// Verify Bob unchanged
	bob := result[1].(map[string]interface{})
	assert.Equal(t, "Bob", bob["name"])
	assert.Equal(t, "2", bob["id"])
}

func TestDeleteBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]`)

	err := Delete(tmpFile, "", "id", "1")
	require.NoError(t, err)

	// Verify only Bob remains
	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 1)

	remaining := result[0].(map[string]interface{})
	assert.Equal(t, "2", remaining["id"])
	assert.Equal(t, "Bob", remaining["name"])
}

func TestDeleteNotFoundBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1","name":"Alice"}]`)

	err := Delete(tmpFile, "", "id", "999")

	require.Error(t, err)
	assert.True(t, IsNotFound(err))

	var notFoundErr *NotFoundError
	require.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "id", notFoundErr.Key)
	assert.Equal(t, "999", notFoundErr.Value)

	// Verify original data unchanged
	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 1)
}

// === Helper Function Tests ===

func TestReadArrayValid(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1"},{"id":"2"}]`)

	result, err := ReadArray(tmpFile)
	require.NoError(t, err)
	require.Len(t, result, 2)

	first := result[0].(map[string]interface{})
	assert.Equal(t, "1", first["id"])
}

func TestReadArrayInvalidObject(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1"}]}`)

	_, err := ReadArray(tmpFile)

	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), "contains a JSON object, but bare-array mode expects a JSON array")
}

func TestWriteArrayRoundTrip(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "roundtrip.json")

	original := []interface{}{
		map[string]interface{}{"id": "1", "name": "Alice"},
		map[string]interface{}{"id": "2", "name": "Bob"},
	}

	err := WriteArray(tmpFile, original)
	require.NoError(t, err)

	result, err := ReadArray(tmpFile)
	require.NoError(t, err)
	require.Len(t, result, 2)

	alice := result[0].(map[string]interface{})
	assert.Equal(t, "1", alice["id"])
	assert.Equal(t, "Alice", alice["name"])
}

func TestEmptyBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[]`)

	incoming := map[string]interface{}{"id": "1", "name": "First"}
	err := Append(tmpFile, "", incoming)
	require.NoError(t, err)

	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 1)

	first := result[0].(map[string]interface{})
	assert.Equal(t, "1", first["id"])
	assert.Equal(t, "First", first["name"])
}

// === Validation Tests ===

func TestValidationMismatchObjectWithoutArrayKey(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1"}]}`)

	incoming := map[string]interface{}{"id": "2"}
	err := Append(tmpFile, "", incoming)

	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), "contains a JSON object but array_key is not specified")
	assert.Contains(t, err.Error(), "Either provide array_key")
}

func TestValidationMismatchArrayWithArrayKey(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1"}]`)

	incoming := map[string]interface{}{"id": "2"}
	err := Append(tmpFile, "items", incoming)

	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), "is a bare JSON array but array_key=\"items\" is specified")
	assert.Contains(t, err.Error(), "Either remove array_key")
}

func TestDetectFileTypeArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1"}]`)

	fileType, err := detectFileType(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "array", fileType)
}

func TestDetectFileTypeObject(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1"}]}`)

	fileType, err := detectFileType(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "object", fileType)
}

func TestDetectFileTypeInvalid(t *testing.T) {
	tmpFile := createTempArrayFile(t, `"just a string"`)

	fileType, err := detectFileType(tmpFile)
	require.Error(t, err)
	assert.Equal(t, "unknown", fileType)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), "must contain either a JSON array or object")
}

// === Backward Compatibility Tests ===

func TestBackwardCompatibilityWithArrayKey(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1","title":"Test"}]}`)

	incoming := map[string]interface{}{
		"id":    "2",
		"title": "New task",
	}

	// Should work exactly as before
	err := Append(tmpFile, "todos", incoming)
	require.NoError(t, err)

	// Verify wrapped structure preserved
	root := readFileAsObject(t, tmpFile)
	todos := root["todos"].([]interface{})
	require.Len(t, todos, 2)

	// Check new record
	newTask := todos[1].(map[string]interface{})
	assert.Equal(t, "2", newTask["id"])
	assert.Equal(t, "New task", newTask["title"])
}

func TestBackwardCompatibilityReplace(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1","title":"Old"},{"id":"2","title":"Keep"}]}`)

	incoming := map[string]interface{}{
		"title": "Updated",
		"done":  true,
	}

	updated, err := Replace(tmpFile, "todos", "id", "1", incoming)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated["title"])
	assert.Equal(t, true, updated["done"])

	// Verify file structure preserved
	root := readFileAsObject(t, tmpFile)
	todos := root["todos"].([]interface{})
	require.Len(t, todos, 2)

	updatedTask := todos[0].(map[string]interface{})
	assert.Equal(t, "Updated", updatedTask["title"])
	assert.Equal(t, true, updatedTask["done"])
	assert.Equal(t, "1", updatedTask["id"]) // preserved

	unchangedTask := todos[1].(map[string]interface{})
	assert.Equal(t, "Keep", unchangedTask["title"])
}

func TestBackwardCompatibilityDelete(t *testing.T) {
	tmpFile := createTempObjectFile(t, `{"todos":[{"id":"1"},{"id":"2"}]}`)

	err := Delete(tmpFile, "todos", "id", "1")
	require.NoError(t, err)

	// Verify wrapped structure preserved
	root := readFileAsObject(t, tmpFile)
	todos := root["todos"].([]interface{})
	require.Len(t, todos, 1)

	remaining := todos[0].(map[string]interface{})
	assert.Equal(t, "2", remaining["id"])
}

// === Enhanced Error Message Tests ===

func TestGetArrayMissingKey(t *testing.T) {
	root := map[string]interface{}{
		"todos": []interface{}{},
		"meta":  map[string]interface{}{},
	}

	_, err := getArray(root, "items")
	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), `array_key "items" not found`)
	assert.Contains(t, err.Error(), `Available keys: ["todos", "meta"]`)
}

func TestGetArrayWrongType(t *testing.T) {
	root := map[string]interface{}{
		"todos": "not an array",
	}

	_, err := getArray(root, "todos")
	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), `field "todos" is not a JSON array`)
	assert.Contains(t, err.Error(), `found string`)
	assert.Contains(t, err.Error(), `"todos": [{...}, {...}]`)
}

func TestGetArrayEmptyArrayKey(t *testing.T) {
	root := map[string]interface{}{
		"todos": []interface{}{},
	}

	_, err := getArray(root, "")
	require.Error(t, err)
	assert.True(t, IsConfigError(err))
	assert.Contains(t, err.Error(), "array_key is required when stub file is a JSON object")
	assert.Contains(t, err.Error(), "For bare JSON arrays, omit array_key entirely")
}

// === Edge Cases ===

func TestReplaceNotFoundBareArray(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1"}]`)

	incoming := map[string]interface{}{"title": "Updated"}
	_, err := Replace(tmpFile, "", "id", "999", incoming)

	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestFileReadError(t *testing.T) {
	nonExistentFile := "/nonexistent/path/file.json"

	err := Append(nonExistentFile, "", map[string]interface{}{"test": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading stub file")
}

func TestInvalidJSONFile(t *testing.T) {
	tmpFile := createTempArrayFile(t, `{"invalid": json}`)

	err := Append(tmpFile, "", map[string]interface{}{"test": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing stub file")
}

func TestNonObjectArrayItems(t *testing.T) {
	tmpFile := createTempArrayFile(t, `[{"id":"1"}, "string", {"id":"2"}]`)

	// Should still work - non-object items are ignored
	incoming := map[string]interface{}{"id": "3", "name": "New"}
	err := Append(tmpFile, "", incoming)
	require.NoError(t, err)

	result := readFileAsArray(t, tmpFile)
	require.Len(t, result, 4) // Original 3 items + 1 new item

	// String item preserved
	assert.Equal(t, "string", result[1])

	// New object added at the end
	newItem := result[3].(map[string]interface{})
	assert.Equal(t, "3", newItem["id"])
}
