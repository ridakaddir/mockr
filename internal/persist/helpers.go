package persist

import (
	"fmt"
	"path/filepath"
	"strings"
)

// evaluateCondition evaluates a simple condition template.
// For now, we'll support basic boolean checks.
func evaluateCondition(condition string, context RequestContext) (bool, error) {
	// Simple template evaluation - check if body contains field
	if strings.Contains(condition, "{{.body.") {
		// Extract field name from {{.body.fieldName}}
		start := strings.Index(condition, "{{.body.") + 8
		end := strings.Index(condition[start:], "}}")
		if end == -1 {
			return false, fmt.Errorf("invalid condition template: %s", condition)
		}
		fieldName := condition[start : start+end]

		// Check if field exists and is truthy in body
		if bodyMap, ok := context.Body.(map[string]interface{}); ok {
			if value, exists := bodyMap[fieldName]; exists {
				// Convert to boolean
				switch v := value.(type) {
				case bool:
					return v, nil
				case string:
					return v != "", nil
				case nil:
					return false, nil
				default:
					return true, nil
				}
			}
		}
		return false, nil
	}

	// Default to true for unsupported conditions for now
	return true, nil
}

// resolveCascadePattern resolves a file pattern to actual file paths.
func resolveCascadePattern(pattern string, context RequestContext) ([]string, error) {
	// First resolve placeholders
	resolvedPattern := resolveFilePath(pattern, context)

	// Handle wildcard patterns
	if strings.Contains(resolvedPattern, "*") {
		matches, err := filepath.Glob(resolvedPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to expand glob pattern %s: %w", resolvedPattern, err)
		}

		// Filter to only .json files for safety
		var jsonFiles []string
		for _, match := range matches {
			if strings.HasSuffix(match, ".json") {
				jsonFiles = append(jsonFiles, match)
			}
		}

		return jsonFiles, nil
	}

	// Single file case
	return []string{resolvedPattern}, nil
}

// applyTransform applies a simple JSONPath-like transformation to data.
func applyTransform(transform string, data interface{}, context RequestContext) (interface{}, error) {
	switch transform {
	case "$":
		// Return entire data
		return data, nil
	case "$.trafficSplit":
		// Extract trafficSplit field
		return extractField(data, "trafficSplit")
	case "$.body.trafficSplit":
		// Extract from request body
		return extractField(context.Body, "trafficSplit")
	default:
		// Handle nested field extraction like "$.field.nested"
		if strings.HasPrefix(transform, "$.") {
			fieldPath := transform[2:] // Remove "$."
			return extractNestedField(data, fieldPath)
		}
		return data, nil
	}
}

// extractField extracts a top-level field from a map.
func extractField(data interface{}, fieldName string) (interface{}, error) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		if value, exists := dataMap[fieldName]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("field %s not found", fieldName)
	}
	return nil, fmt.Errorf("data is not a map")
}

// extractNestedField extracts a nested field using dot notation.
func extractNestedField(data interface{}, fieldPath string) (interface{}, error) {
	parts := strings.Split(fieldPath, ".")
	current := data

	for _, part := range parts {
		if currentMap, ok := current.(map[string]interface{}); ok {
			if value, exists := currentMap[part]; exists {
				current = value
			} else {
				return nil, fmt.Errorf("field %s not found in path %s", part, fieldPath)
			}
		} else {
			return nil, fmt.Errorf("cannot traverse non-map value at %s", part)
		}
	}

	return current, nil
}

// notifyWatchers notifies file watchers about changes.
func notifyWatchers(filePaths []string) {
	// Notify file watchers of updates
	for _, path := range filePaths {
		notify(path, FileUpdated)
	}
}
