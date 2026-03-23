package proxy

import (
	"regexp"
	"strings"
)

// matchPath returns true if the incoming request path matches the route's match pattern.
// Four match styles are supported:
//
//	exact:    "/api/users"           — must equal path exactly
//	wildcard: "/api/users/*"         — prefix match; * matches any single segment or multiple segments
//	named:    "/api/users/{userId}"  — {name} matches exactly one path segment
//	regex:    "~^/api/users/\d+$"    — full regexp (prefix with ~)
func matchPath(pattern, path string) bool {
	matched, _ := matchWithNamedParams(pattern, path)
	return matched
}

// matchWildcard performs simple glob-style matching where * matches any sequence
// of non-empty characters (including path separators).
func matchWildcard(pattern, path string) bool {
	// Split on * and ensure each part appears in order inside path.
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return true
	}

	remaining := path
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(remaining, part)
		if idx == -1 {
			return false
		}
		// The first part must be a prefix.
		if i == 0 && idx != 0 {
			return false
		}
		remaining = remaining[idx+len(part):]
	}

	// If the pattern doesn't end with *, the path must be fully consumed.
	if !strings.HasSuffix(pattern, "*") && remaining != "" {
		return false
	}

	return true
}

// extractWildcardValue extracts the segment of path that corresponds to the
// first * in pattern. Used by persist.go to resolve the key value from the URL.
// Returns ("", false) if no wildcard or no match.
func extractWildcardValue(pattern, path string) (string, bool) {
	if !strings.Contains(pattern, "*") {
		return "", false
	}

	star := strings.Index(pattern, "*")
	prefix := pattern[:star]
	suffix := pattern[star+1:]

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	after := path[len(prefix):]

	if suffix == "" {
		// * is at the end — value is everything after prefix
		// but stop at the next /
		if idx := strings.Index(after, "/"); idx != -1 {
			return after[:idx], true
		}
		return after, true
	}

	idx := strings.LastIndex(after, suffix)
	if idx == -1 {
		return "", false
	}

	return after[:idx], true
}

// hasNamedParams returns true if the pattern contains {name} placeholders.
func hasNamedParams(pattern string) bool {
	return strings.Contains(pattern, "{") && strings.Contains(pattern, "}")
}

// extractNamedParams extracts named parameters from a path using segment-by-segment matching.
// Returns a map of parameter names to values and whether the pattern matched.
// Example: pattern="/api/{version}/users/{userId}" path="/api/v1/users/123"
// returns map[string]string{"version": "v1", "userId": "123"}, true
func extractNamedParams(pattern, path string) (map[string]string, bool) {
	if !hasNamedParams(pattern) {
		return nil, false
	}

	// Split both pattern and path by '/'
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	params := make(map[string]string)
	pathIndex := 0

	for i, patternPart := range patternParts {
		// Handle empty parts (from leading/trailing slashes)
		if patternPart == "" {
			continue
		}

		// Check if we've consumed all path parts
		if pathIndex >= len(pathParts) {
			return nil, false
		}

		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			// Named parameter: extract name and capture value
			paramName := patternPart[1 : len(patternPart)-1]
			if paramName == "" {
				return nil, false // Invalid empty parameter name
			}
			params[paramName] = pathParts[pathIndex]
			pathIndex++
		} else if patternPart == "*" {
			// Wildcard: consume one or more segments
			if i == len(patternParts)-1 {
				// * at end: consume all remaining segments
				pathIndex = len(pathParts)
			} else {
				// * in middle: consume until next literal segment matches
				nextPattern := patternParts[i+1]
				found := false
				for pathIndex < len(pathParts) {
					if pathParts[pathIndex] == nextPattern {
						found = true
						break
					}
					pathIndex++
				}
				if !found {
					return nil, false
				}
			}
		} else {
			// Literal segment: must match exactly
			if pathParts[pathIndex] != patternPart {
				return nil, false
			}
			pathIndex++
		}
	}

	// Ensure all path segments were consumed
	if pathIndex != len(pathParts) {
		return nil, false
	}

	return params, true
}

// matchWithNamedParams performs path matching and returns both the match result
// and any extracted named parameters.
func matchWithNamedParams(pattern, path string) (bool, map[string]string) {
	if strings.HasPrefix(pattern, "~") {
		// Regex match - no named params support
		re, err := regexp.Compile(pattern[1:])
		if err != nil {
			return false, nil
		}
		return re.MatchString(path), nil
	}

	if hasNamedParams(pattern) {
		// Named parameters matching
		params, matched := extractNamedParams(pattern, path)
		return matched, params
	}

	if strings.Contains(pattern, "*") {
		// Existing wildcard matching
		return matchWildcard(pattern, path), nil
	}

	// Exact match
	return pattern == path, nil
}
