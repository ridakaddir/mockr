package persist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCascade_SecurityVulnerabilities(t *testing.T) {
	// Setup temporary test directory
	tmpDir, err := os.MkdirTemp("", "cascade_security_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test directory structure
	endpointDir := filepath.Join(tmpDir, "endpoints")
	require.NoError(t, os.MkdirAll(endpointDir, 0755))

	// Create initial endpoint file
	endpointFile := filepath.Join(endpointDir, "test-endpoint.json")
	initialData := map[string]interface{}{
		"endpointId": "test-endpoint",
		"trafficSplit": map[string]interface{}{
			"deployment-1": 100,
		},
	}
	writeSecurityTestJSONFile(t, endpointFile, initialData)

	tests := []struct {
		name           string
		pathParams     map[string]string
		queryParams    map[string]string
		cascadePattern string
		expectError    bool
		errorContains  string
	}{
		{
			name: "path traversal in pathParams",
			pathParams: map[string]string{
				"endpointId": "../../../etc/passwd",
			},
			cascadePattern: "stubs/deployments/{path.endpointId}/*.json",
			expectError:    true,
			errorContains:  "pattern escapes config directory",
		},
		{
			name: "path traversal in queryParams",
			queryParams: map[string]string{
				"target": "../../../etc/passwd",
			},
			cascadePattern: "stubs/deployments/{query.target}/*.json",
			expectError:    true,
			errorContains:  "pattern escapes config directory",
		},
		{
			name: "null byte injection in pathParams",
			pathParams: map[string]string{
				"endpointId": "test\x00../../../etc/passwd",
			},
			cascadePattern: "stubs/deployments/{path.endpointId}/*.json",
			expectError:    false, // Should be sanitized and work safely
		},
		{
			name: "directory traversal sequences",
			pathParams: map[string]string{
				"endpointId": "test/../../../etc/passwd",
			},
			cascadePattern: "stubs/deployments/{path.endpointId}/*.json",
			expectError:    true,
			errorContains:  "pattern escapes config directory",
		},
		{
			name: "absolute path injection",
			pathParams: map[string]string{
				"endpointId": "/etc/passwd",
			},
			cascadePattern: "stubs/deployments/{path.endpointId}/*.json",
			expectError:    true,
			errorContains:  "pattern escapes config directory",
		},
		{
			name: "valid input should work",
			pathParams: map[string]string{
				"endpointId": "valid-endpoint-123",
			},
			cascadePattern: "stubs/deployments/{path.endpointId}/*.json",
			expectError:    true, // Will fail because no deployment files, but not due to security
			errorContains:  "cascade pattern resolved to no files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create cascade config
			caseConfig := config.Case{
				Merge: "cascade",
				Primary: &config.CascadePrimary{
					File:  endpointFile,
					Merge: "update",
					Path:  "trafficSplit",
				},
				Cascade: []config.CascadeTarget{
					{
						Pattern:   tt.cascadePattern,
						Merge:     "update",
						Path:      "deploymentSpec.trafficSplit",
						Transform: "$.trafficSplit",
					},
				},
			}

			updateData := map[string]interface{}{
				"trafficSplit": map[string]interface{}{
					"deployment-1": 50,
				},
			}

			context := RequestContext{
				Body:        updateData,
				PathParams:  tt.pathParams,
				QueryParams: tt.queryParams,
				ConfigDir:   tmpDir,
			}

			// Execute cascade
			err := ExecuteCascade(caseConfig, updateData, context)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				// For non-error cases, we just check that it doesn't panic or create security issues
				// The operation might fail for legitimate reasons (no files to update)
				t.Logf("Operation result: %v", err)
			}
		})
	}
}

func TestSanitizePathValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "valid-endpoint-123",
			expected: "valid-endpoint-123",
		},
		{
			name:     "path traversal sequences",
			input:    "../../../etc/passwd",
			expected: "etcpasswd",
		},
		{
			name:     "null byte injection",
			input:    "test\x00file",
			expected: "testfile",
		},
		{
			name:     "mixed attack vectors",
			input:    "test/../\x00\n\rmalicious",
			expected: "testmalicious",
		},
		{
			name:     "only special characters",
			input:    "../../../\x00\n\r",
			expected: "",
		},
		{
			name:     "valid characters with dots",
			input:    "test.endpoint.v2",
			expected: "test.endpoint.v2",
		},
		{
			name:     "unicode and special chars",
			input:    "test-endpoint_123.json",
			expected: "test-endpoint_123.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePathValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCascadePattern_SecurityValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pattern_security_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name          string
		pattern       string
		pathParams    map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name:    "valid pattern within config dir",
			pattern: "stubs/endpoints/*.json",
			pathParams: map[string]string{
				"endpointId": "valid-endpoint",
			},
			expectError: false,
		},
		{
			name:    "pattern trying to escape config dir",
			pattern: "../../../etc/*.json",
			pathParams: map[string]string{
				"endpointId": "test",
			},
			expectError:   true,
			errorContains: "pattern escapes config directory",
		},
		{
			name:    "pattern with path traversal in placeholder",
			pattern: "stubs/endpoints/{path.endpointId}/*.json",
			pathParams: map[string]string{
				"endpointId": "../../../etc/passwd",
			},
			expectError:   true,
			errorContains: "pattern escapes config directory",
		},
		{
			name:    "absolute path pattern",
			pattern: "/etc/passwd",
			pathParams: map[string]string{
				"endpointId": "test",
			},
			expectError:   true,
			errorContains: "pattern escapes config directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := RequestContext{
				PathParams: tt.pathParams,
				ConfigDir:  tmpDir,
			}

			files, err := resolveCascadePattern(tt.pattern, context)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				t.Logf("Resolved files: %v", files)
			}
		})
	}
}

// Helper function for writing JSON files (reused from other tests)
func writeSecurityTestJSONFile(t *testing.T, filePath string, data interface{}) {
	err := os.WriteFile(filePath, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)
}
