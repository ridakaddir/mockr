package proxy

import (
	"reflect"
	"testing"
)

func TestHasNamedParams(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"/api/users", false},
		{"/api/users/*", false},
		{"/api/users/{id}", true},
		{"/api/{version}/users/{id}", true},
		{"/api/v1/environments/{envId}/endpoint/{endpointId}", true},
		{"", false},
		{"{", false},                  // Invalid
		{"}", false},                  // Invalid
		{"{incomplete", false},        // Invalid
		{"incomplete}", false},        // Invalid
		{"~^/api/users/\\d+$", false}, // Regex pattern
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := hasNamedParams(tt.pattern)
			if got != tt.want {
				t.Errorf("hasNamedParams(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestExtractNamedParams(t *testing.T) {
	tests := []struct {
		pattern    string
		path       string
		wantParams map[string]string
		wantMatch  bool
	}{
		// Basic named parameter cases
		{
			pattern:    "/api/users/{id}",
			path:       "/api/users/123",
			wantParams: map[string]string{"id": "123"},
			wantMatch:  true,
		},
		{
			pattern:    "/api/{version}/users/{userId}",
			path:       "/api/v1/users/456",
			wantParams: map[string]string{"version": "v1", "userId": "456"},
			wantMatch:  true,
		},
		// Complex example from spec
		{
			pattern:    "/api/v1/*/environments/{envId}/endpoint/{endpointId}",
			path:       "/api/v1/staging/environments/mcp-test/endpoint/920316521215950848",
			wantParams: map[string]string{"envId": "mcp-test", "endpointId": "920316521215950848"},
			wantMatch:  true,
		},
		// Mixed wildcards and named params
		{
			pattern:    "/api/*/users/{userId}/posts/*",
			path:       "/api/v1/users/123/posts/456",
			wantParams: map[string]string{"userId": "123"},
			wantMatch:  true,
		},
		// Test wildcard followed by named parameter (reported by Copilot)
		{
			pattern:    "/api/*/{userId}",
			path:       "/api/v1/123",
			wantParams: map[string]string{"userId": "123"},
			wantMatch:  true,
		},
		{
			pattern:    "/api/*/{userId}/posts/{postId}",
			path:       "/api/v1/v2/123/posts/456",
			wantParams: map[string]string{"userId": "123", "postId": "456"},
			wantMatch:  true,
		},
		// Test multiple wildcards
		{
			pattern:    "/api/*/*/{userId}",
			path:       "/api/v1/staging/123",
			wantParams: map[string]string{"userId": "123"},
			wantMatch:  true,
		},
		// No named params in pattern
		{
			pattern:    "/api/users",
			path:       "/api/users",
			wantParams: nil,
			wantMatch:  false,
		},
		// Mismatch cases
		{
			pattern:    "/api/users/{id}",
			path:       "/api/posts/123",
			wantParams: nil,
			wantMatch:  false,
		},
		{
			pattern:    "/api/{version}/users/{userId}",
			path:       "/api/v1/users", // Missing userId segment
			wantParams: nil,
			wantMatch:  false,
		},
		{
			pattern:    "/api/{version}/users/{userId}",
			path:       "/api/v1/users/123/extra", // Extra segments
			wantParams: nil,
			wantMatch:  false,
		},
		// Edge cases
		{
			pattern:    "/{single}",
			path:       "/test",
			wantParams: map[string]string{"single": "test"},
			wantMatch:  true,
		},
		{
			pattern:    "",
			path:       "",
			wantParams: nil,
			wantMatch:  false,
		},
		{
			pattern:    "/api/{}", // Empty param name
			path:       "/api/test",
			wantParams: nil,
			wantMatch:  false,
		},
		// Leading/trailing slash variations
		{
			pattern:    "api/users/{id}",
			path:       "api/users/123",
			wantParams: map[string]string{"id": "123"},
			wantMatch:  true,
		},
		{
			pattern:    "/api/users/{id}/",
			path:       "/api/users/123/",
			wantParams: map[string]string{"id": "123"},
			wantMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			gotParams, gotMatch := extractNamedParams(tt.pattern, tt.path)

			if gotMatch != tt.wantMatch {
				t.Errorf("extractNamedParams(%q, %q) match = %v, want %v",
					tt.pattern, tt.path, gotMatch, tt.wantMatch)
			}

			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("extractNamedParams(%q, %q) params = %v, want %v",
					tt.pattern, tt.path, gotParams, tt.wantParams)
			}
		})
	}
}

func TestMatchWithNamedParams(t *testing.T) {
	tests := []struct {
		pattern    string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		// Named parameters
		{
			pattern:    "/api/users/{id}",
			path:       "/api/users/123",
			wantMatch:  true,
			wantParams: map[string]string{"id": "123"},
		},
		// Existing wildcard behavior (should work unchanged)
		{
			pattern:    "/api/users/*",
			path:       "/api/users/123",
			wantMatch:  true,
			wantParams: nil,
		},
		// Exact match (should work unchanged)
		{
			pattern:    "/api/users",
			path:       "/api/users",
			wantMatch:  true,
			wantParams: nil,
		},
		// Regex pattern (should work unchanged)
		{
			pattern:    "~^/api/users/\\d+$",
			path:       "/api/users/123",
			wantMatch:  true,
			wantParams: nil,
		},
		// Regex pattern that doesn't match
		{
			pattern:    "~^/api/users/\\d+$",
			path:       "/api/users/abc",
			wantMatch:  false,
			wantParams: nil,
		},
		// Invalid regex
		{
			pattern:    "~[invalid",
			path:       "/api/users",
			wantMatch:  false,
			wantParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			gotMatch, gotParams := matchWithNamedParams(tt.pattern, tt.path)

			if gotMatch != tt.wantMatch {
				t.Errorf("matchWithNamedParams(%q, %q) match = %v, want %v",
					tt.pattern, tt.path, gotMatch, tt.wantMatch)
			}

			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("matchWithNamedParams(%q, %q) params = %v, want %v",
					tt.pattern, tt.path, gotParams, tt.wantParams)
			}
		})
	}
}
