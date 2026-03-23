package proxy

import (
	"net/http"
	"regexp"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{(\w+)\.([^}]+)\}`)

// resolveDynamicFile replaces {source.field} placeholders in a file path
// with values extracted from the current request.
//
// Examples:
//
//	"stubs/user-{query.username}-orders.json"  → "stubs/user-john-orders.json"
//	"stubs/{body.user.id}-profile.json"        → "stubs/42-profile.json"
//	"stubs/{header.X-User-Id}-data.json"       → "stubs/abc-data.json"
//	"stubs/endpoint-{path.endpointId}.json"    → "stubs/endpoint-12345.json"
func resolveDynamicFile(filePath string, r *http.Request, bodyBytes []byte, routePattern string, pathParams map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(filePath, func(match string) string {
		sub := placeholderRe.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		source := strings.ToLower(sub[1])
		field := sub[2]

		val, found := extractValue(source, field, r, bodyBytes, routePattern, pathParams)
		if !found {
			return match // leave placeholder as-is if not resolved
		}
		return sanitizePathSegment(val)
	})
}

// hasDynamicPlaceholders returns true if the file path contains any {source.field} tokens.
func hasDynamicPlaceholders(filePath string) bool {
	return placeholderRe.MatchString(filePath)
}

// sanitizePathSegment strips characters that are unsafe in file names.
// Importantly, it prevents directory traversal by replacing . and .. with safe alternatives.
func sanitizePathSegment(s string) string {
	// First, handle exact directory traversal attempts
	if s == "." || s == ".." {
		return "_"
	}

	// Strip unsafe characters, but preserve dots for normal file extensions
	unsafe := regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)
	sanitized := unsafe.ReplaceAllString(s, "_")

	// Prevent leading dots which could create hidden files or relative paths
	if strings.HasPrefix(sanitized, ".") {
		sanitized = "_" + sanitized[1:]
	}

	return sanitized
}
