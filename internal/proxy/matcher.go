package proxy

import (
	"regexp"
	"strings"
)

// matchPath returns true if the incoming request path matches the route's match pattern.
// Three match styles are supported:
//
//	exact:    "/api/users"        — must equal path exactly
//	wildcard: "/api/users/*"      — prefix match; * matches any single segment or multiple segments
//	regex:    "~^/api/users/\d+$" — full regexp (prefix with ~)
func matchPath(pattern, path string) bool {
	if strings.HasPrefix(pattern, "~") {
		// Regex match
		re, err := regexp.Compile(pattern[1:])
		if err != nil {
			return false
		}
		return re.MatchString(path)
	}

	if strings.Contains(pattern, "*") {
		return matchWildcard(pattern, path)
	}

	// Exact match
	return pattern == path
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
