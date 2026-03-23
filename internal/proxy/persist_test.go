package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestResolveKeyValueWithNamedParams(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		pathParams   map[string]string
		routePattern string
		requestPath  string
		body         string
		queryParams  map[string]string
		want         string
	}{
		{
			name:       "named parameter takes priority",
			key:        "userId",
			pathParams: map[string]string{"userId": "123", "postId": "456"},
			want:       "123",
		},
		{
			name:         "fallback to wildcard when named param not found",
			key:          "id",
			pathParams:   map[string]string{"userId": "123"},
			routePattern: "/api/users/*",
			requestPath:  "/api/users/wildcard-value",
			want:         "wildcard-value",
		},
		{
			name:       "fallback to body when no path params",
			key:        "userId",
			pathParams: nil,
			body:       `{"userId": "body-value"}`,
			want:       "body-value",
		},
		{
			name:        "fallback to query when no other sources",
			key:         "userId",
			pathParams:  nil,
			queryParams: map[string]string{"userId": "query-value"},
			want:        "query-value",
		},
		{
			name:       "empty named param falls back to next source",
			key:        "userId",
			pathParams: map[string]string{"userId": ""},
			body:       `{"userId": "body-value"}`,
			want:       "body-value",
		},
		{
			name:        "named param wins over all other sources",
			key:         "userId",
			pathParams:  map[string]string{"userId": "named-value"},
			body:        `{"userId": "body-value"}`,
			queryParams: map[string]string{"userId": "query-value"},
			want:        "named-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock request
			req := &http.Request{
				URL: &url.URL{
					Path: tt.requestPath,
				},
			}

			// Add query parameters
			if tt.queryParams != nil {
				q := url.Values{}
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			bodyBytes := []byte(tt.body)

			got := resolveKeyValue(tt.key, req, bodyBytes, tt.routePattern, tt.pathParams)
			if got != tt.want {
				t.Errorf("resolveKeyValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
