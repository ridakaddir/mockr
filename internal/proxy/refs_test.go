package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRefs_NoRefs(t *testing.T) {
	content := []byte(`{"id": "123", "name": "test"}`)
	result, err := resolveRefs(content, "", make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(content) {
		t.Errorf("content should remain unchanged when no refs present")
	}
}

func TestResolveRefs_Directory(t *testing.T) {
	// Setup temporary directory with test files
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	// Create test files
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice"}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}
	file2 := filepath.Join(dataDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"id": "2", "name": "Bob"}`), 0644); err != nil {
		t.Fatalf("writing test file 2: %v", err)
	}

	// Test directory reference
	content := []byte(`{"users": "{{ref:data/}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	users, ok := parsed["users"].([]interface{})
	if !ok {
		t.Fatalf("users should be an array")
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestResolveRefs_SingleFile(t *testing.T) {
	// Setup temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "user.json")
	if err := os.WriteFile(testFile, []byte(`{"id": "1", "name": "Alice"}`), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	// Test single file reference
	content := []byte(`{"user": "{{ref:user.json}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	user, ok := parsed["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("user should be an object")
	}
	if user["id"] != "1" || user["name"] != "Alice" {
		t.Errorf("unexpected user data: %v", user)
	}
}

func TestResolveRefs_WithFilter(t *testing.T) {
	// Setup temporary directory with test files
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "users")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	// Create test files with different statuses
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice", "status": "active"}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}
	file2 := filepath.Join(dataDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"id": "2", "name": "Bob", "status": "inactive"}`), 0644); err != nil {
		t.Fatalf("writing test file 2: %v", err)
	}
	file3 := filepath.Join(dataDir, "3.json")
	if err := os.WriteFile(file3, []byte(`{"id": "3", "name": "Charlie", "status": "active"}`), 0644); err != nil {
		t.Fatalf("writing test file 3: %v", err)
	}

	// Test filter reference
	content := []byte(`{"activeUsers": "{{ref:users/?filter=status:active}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	users, ok := parsed["activeUsers"].([]interface{})
	if !ok {
		t.Fatalf("activeUsers should be an array")
	}
	if len(users) != 2 {
		t.Errorf("expected 2 active users, got %d", len(users))
	}

	// Check that both active users are present
	names := make(map[string]bool)
	for _, user := range users {
		u := user.(map[string]interface{})
		names[u["name"].(string)] = true
		if u["status"] != "active" {
			t.Errorf("filtered user should have status 'active', got %v", u["status"])
		}
	}
	if !names["Alice"] || !names["Charlie"] {
		t.Errorf("expected Alice and Charlie, got users: %v", names)
	}
}

func TestResolveRefs_WithNestedFilter(t *testing.T) {
	// Setup temporary directory with test files containing nested objects
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "users")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	// Create test files with nested user data
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice", "user": {"role": "admin"}}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}
	file2 := filepath.Join(dataDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"id": "2", "name": "Bob", "user": {"role": "user"}}`), 0644); err != nil {
		t.Fatalf("writing test file 2: %v", err)
	}

	// Test nested filter reference
	content := []byte(`{"admins": "{{ref:users/?filter=user.role:admin}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	admins, ok := parsed["admins"].([]interface{})
	if !ok {
		t.Fatalf("admins should be an array")
	}
	if len(admins) != 1 {
		t.Errorf("expected 1 admin, got %d", len(admins))
	}

	admin := admins[0].(map[string]interface{})
	if admin["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", admin["name"])
	}
}

func TestResolveRefs_WithTemplate(t *testing.T) {
	// Setup temporary directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "users")
	templatesDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating data directory: %v", err)
	}
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("creating templates directory: %v", err)
	}

	// Create test files
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice", "email": "alice@example.com"}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}
	file2 := filepath.Join(dataDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"id": "2", "name": "Bob", "email": "bob@example.com"}`), 0644); err != nil {
		t.Fatalf("writing test file 2: %v", err)
	}

	// Create template file
	templateFile := filepath.Join(templatesDir, "user-summary.json")
	templateContent := `{"userId": "{{.id}}", "displayName": "{{.name}}"}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("writing template file: %v", err)
	}

	// Test template reference
	content := []byte(`{"users": "{{ref:users/?template=templates/user-summary.json}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	users, ok := parsed["users"].([]interface{})
	if !ok {
		t.Fatalf("users should be an array")
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Check transformed structure
	for _, user := range users {
		u := user.(map[string]interface{})
		if _, hasUserId := u["userId"]; !hasUserId {
			t.Errorf("user should have userId field after template transformation")
		}
		if _, hasDisplayName := u["displayName"]; !hasDisplayName {
			t.Errorf("user should have displayName field after template transformation")
		}
		if _, hasEmail := u["email"]; hasEmail {
			t.Errorf("user should not have email field after template transformation")
		}
	}
}

func TestResolveRefs_FilterAndTemplate(t *testing.T) {
	// Setup temporary directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "users")
	templatesDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating data directory: %v", err)
	}
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("creating templates directory: %v", err)
	}

	// Create test files
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice", "status": "active"}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}
	file2 := filepath.Join(dataDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"id": "2", "name": "Bob", "status": "inactive"}`), 0644); err != nil {
		t.Fatalf("writing test file 2: %v", err)
	}

	// Create template file
	templateFile := filepath.Join(templatesDir, "active-user.json")
	templateContent := `{"userId": "{{.id}}", "userName": "{{.name}}"}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("writing template file: %v", err)
	}

	// Test filter + template reference
	content := []byte(`{"activeUsers": "{{ref:users/?filter=status:active&template=templates/active-user.json}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	users, ok := parsed["activeUsers"].([]interface{})
	if !ok {
		t.Fatalf("activeUsers should be an array")
	}
	if len(users) != 1 {
		t.Errorf("expected 1 active user after filter, got %d", len(users))
	}

	// Check filtered and transformed user
	user := users[0].(map[string]interface{})
	if user["userId"] != "1" || user["userName"] != "Alice" {
		t.Errorf("unexpected user data after filter+template: %v", user)
	}
	if _, hasStatus := user["status"]; hasStatus {
		t.Errorf("user should not have status field after template transformation")
	}
}

func TestResolveRefs_EmptyFilterResult(t *testing.T) {
	// Setup temporary directory with test files
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "users")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	// Create test files (none matching the filter)
	file1 := filepath.Join(dataDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"id": "1", "name": "Alice", "status": "inactive"}`), 0644); err != nil {
		t.Fatalf("writing test file 1: %v", err)
	}

	// Test filter that matches nothing
	content := []byte(`{"activeUsers": "{{ref:users/?filter=status:active}}"}`)
	result, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("parsing result: %v", err)
	}

	users, ok := parsed["activeUsers"].([]interface{})
	if !ok {
		t.Fatalf("activeUsers should be an array")
	}
	if len(users) != 0 {
		t.Errorf("expected empty array when filter matches nothing, got %d items", len(users))
	}
}

func TestResolveRefs_MissingFile(t *testing.T) {
	tempDir := t.TempDir()
	content := []byte(`{"data": "{{ref:nonexistent.json}}"}`)

	_, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "nonexistent.json") {
		t.Errorf("error should mention missing file, got: %v", err)
	}
}

func TestResolveRefs_CircularDetection(t *testing.T) {
	// Setup temporary files with circular references
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "1.json")
	if err := os.WriteFile(file1, []byte(`{"ref": "{{ref:2.json}}"}`), 0644); err != nil {
		t.Fatalf("writing file 1: %v", err)
	}
	file2 := filepath.Join(tempDir, "2.json")
	if err := os.WriteFile(file2, []byte(`{"ref": "{{ref:1.json}}"}`), 0644); err != nil {
		t.Fatalf("writing file 2: %v", err)
	}

	content := []byte(`{"data": "{{ref:1.json}}"}`)

	_, err := resolveRefs(content, tempDir, make(map[string]bool))
	if err == nil {
		t.Fatalf("expected error for circular reference")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error should mention circular reference, got: %v", err)
	}
}

func TestParseRefToken_Simple(t *testing.T) {
	path, filters, templatePath, err := parseRefToken("path/to/dir/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "path/to/dir/" {
		t.Errorf("expected path 'path/to/dir/', got '%s'", path)
	}
	if len(filters) != 0 {
		t.Errorf("expected no filters, got %d", len(filters))
	}
	if templatePath != "" {
		t.Errorf("expected no template path, got '%s'", templatePath)
	}
}

func TestParseRefToken_WithFilter(t *testing.T) {
	path, filters, templatePath, err := parseRefToken("path/?filter=status:active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "path/" {
		t.Errorf("expected path 'path/', got '%s'", path)
	}
	if len(filters) != 1 || filters["status"] != "active" {
		t.Errorf("expected filter status:active, got %v", filters)
	}
	if templatePath != "" {
		t.Errorf("expected no template path, got '%s'", templatePath)
	}
}

func TestParseRefToken_WithTemplate(t *testing.T) {
	path, filters, templatePath, err := parseRefToken("path/?template=tpl.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "path/" {
		t.Errorf("expected path 'path/', got '%s'", path)
	}
	if len(filters) != 0 {
		t.Errorf("expected no filters, got %d", len(filters))
	}
	if templatePath != "tpl.json" {
		t.Errorf("expected template path 'tpl.json', got '%s'", templatePath)
	}
}

func TestParseRefToken_WithBoth(t *testing.T) {
	path, filters, templatePath, err := parseRefToken("path/?filter=status:active&template=tpl.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "path/" {
		t.Errorf("expected path 'path/', got '%s'", path)
	}
	if len(filters) != 1 || filters["status"] != "active" {
		t.Errorf("expected filter status:active, got %v", filters)
	}
	if templatePath != "tpl.json" {
		t.Errorf("expected template path 'tpl.json', got '%s'", templatePath)
	}
}

func TestParseRefToken_DuplicateFilter(t *testing.T) {
	_, _, _, err := parseRefToken("path/?filter=status:active&filter=status:inactive")
	if err == nil {
		t.Fatalf("expected error for duplicate filter")
	}
	if !strings.Contains(err.Error(), "duplicate filter") {
		t.Errorf("error should mention duplicate filter, got: %v", err)
	}
}

func TestParseRefToken_InvalidFilter(t *testing.T) {
	_, _, _, err := parseRefToken("path/?filter=invalid")
	if err == nil {
		t.Fatalf("expected error for invalid filter format")
	}
	if !strings.Contains(err.Error(), "invalid filter format") {
		t.Errorf("error should mention invalid filter format, got: %v", err)
	}
}
