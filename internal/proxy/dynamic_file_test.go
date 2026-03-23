package proxy

import (
	"testing"
)

func TestSanitizePathSegment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Normal safe inputs
		{"user123", "user123"},
		{"test-file", "test-file"},
		{"file_name", "file_name"},
		{"data.json", "data.json"},
		{"user-123_data.txt", "user-123_data.txt"},

		// Directory traversal attempts
		{"..", "_"},
		{".", "_"},

		// Leading dots (hidden files / relative paths)
		{".hidden", "_hidden"},
		{"..secret", "_.secret"},

		// Unsafe characters
		{"user@123", "user_123"},
		{"file/path", "file_path"},
		{"user\\123", "user_123"},
		{"test:file", "test_file"},
		{"file*name", "file_name"},
		{"file?name", "file_name"},
		{"file<name>", "file_name_"},
		{"file|name", "file_name"},

		// Mixed unsafe and directory traversal
		{"../secret", "_._secret"},
		{"./hidden", "__hidden"},

		// Empty string
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizePathSegment(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePathSegment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizePathSegmentDirectoryTraversalPrevention(t *testing.T) {
	// Test that common directory traversal patterns are neutralized
	dangerous := []string{"..", ".", "../", "./"}

	for _, input := range dangerous {
		result := sanitizePathSegment(input)

		// Should not contain any dots that could be used for traversal
		if result == ".." || result == "." || result == "../" || result == "./" {
			t.Errorf("sanitizePathSegment(%q) = %q, should neutralize directory traversal", input, result)
		}

		// Should not start with dots
		if len(result) > 0 && result[0] == '.' {
			t.Errorf("sanitizePathSegment(%q) = %q, should not start with dot", input, result)
		}
	}
}
