package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ridakaddir/mockr/internal/persist"
)

// refPattern matches "{{ref:path?params}}" tokens (including surrounding quotes)
var refPattern = regexp.MustCompile(`"{{ref:([^}]+)}}"`)

// resolveRefs processes all {{ref:...}} tokens in JSON content.
// visited tracks paths to detect circular references.
func resolveRefs(content []byte, configDir string, visited map[string]bool) ([]byte, error) {
	// Skip processing if content is empty or whitespace-only
	if len(content) == 0 || len(strings.TrimSpace(string(content))) == 0 {
		return content, nil
	}

	// Find all ref tokens
	matches := refPattern.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Process matches in reverse order to preserve indices
	result := content
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		token := string(content[match[2]:match[3]]) // path?params

		// Resolve this reference
		resolved, err := resolveRefToken(token, configDir, visited)
		if err != nil {
			return nil, fmt.Errorf("resolving ref %q: %w", token, err)
		}

		// Marshal resolved data to JSON
		jsonBytes, err := json.Marshal(resolved)
		if err != nil {
			return nil, fmt.Errorf("marshalling ref result: %w", err)
		}

		// Replace the token (including quotes) with the JSON value using a new buffer
		prefix := result[:match[0]]
		suffix := result[match[1]:]
		buf := make([]byte, 0, len(prefix)+len(jsonBytes)+len(suffix))
		buf = append(buf, prefix...)
		buf = append(buf, jsonBytes...)
		buf = append(buf, suffix...)
		result = buf
	}

	// Validate that the result is still valid JSON after replacements
	var validation interface{}
	if err := json.Unmarshal(result, &validation); err != nil {
		return nil, fmt.Errorf("invalid JSON after ref resolution: %w", err)
	}

	return result, nil
}

// resolveRefToken parses and resolves a single ref token
func resolveRefToken(token string, configDir string, visited map[string]bool) (interface{}, error) {
	// Parse token: path?filter=x:y&template=z
	path, filters, templatePath, err := parseRefToken(token)
	if err != nil {
		return nil, err
	}

	// Prevent directory traversal and absolute path attacks
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("absolute paths not allowed in ref tokens: %s", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("directory traversal not allowed in ref tokens: %s", path)
	}

	// Preserve directory intent before path resolution
	isDirectoryIntent := strings.HasSuffix(path, "/")

	// Resolve relative path within configDir
	absPath := path
	if configDir != "" {
		absPath = filepath.Join(configDir, path)

		// Ensure resolved path stays within configDir
		absConfig, err := filepath.Abs(configDir)
		if err != nil {
			return nil, fmt.Errorf("resolving config directory: %w", err)
		}
		absResolved, err := filepath.Abs(absPath)
		if err != nil {
			return nil, fmt.Errorf("resolving reference path: %w", err)
		}
		if !strings.HasPrefix(absResolved, absConfig+string(filepath.Separator)) && absResolved != absConfig {
			return nil, fmt.Errorf("reference path escapes config directory: %s", path)
		}
	}

	// Check for circular reference
	if visited[absPath] {
		return nil, fmt.Errorf("circular reference detected: %s", absPath)
	}
	visited[absPath] = true
	defer delete(visited, absPath)

	// Load data from file or directory
	data, err := loadRefData(absPath, isDirectoryIntent)
	if err != nil {
		return nil, err
	}

	// Recursively resolve any nested refs in the loaded data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	dataBytes, err = resolveRefs(dataBytes, configDir, visited)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, err
	}

	// Apply filters if specified
	if len(filters) > 0 {
		data, err = filterData(data, filters)
		if err != nil {
			return nil, err
		}
	}

	// Apply template if specified
	if templatePath != "" {
		// Prevent directory traversal and absolute path attacks for templates
		if filepath.IsAbs(templatePath) {
			return nil, fmt.Errorf("absolute paths not allowed in template tokens: %s", templatePath)
		}
		if strings.Contains(templatePath, "..") {
			return nil, fmt.Errorf("directory traversal not allowed in template tokens: %s", templatePath)
		}

		absTemplatePath := templatePath
		if configDir != "" {
			absTemplatePath = filepath.Join(configDir, templatePath)

			// Ensure template path stays within configDir
			absConfig, err := filepath.Abs(configDir)
			if err != nil {
				return nil, fmt.Errorf("resolving config directory for template: %w", err)
			}
			absTemplate, err := filepath.Abs(absTemplatePath)
			if err != nil {
				return nil, fmt.Errorf("resolving template path: %w", err)
			}
			if !strings.HasPrefix(absTemplate, absConfig+string(filepath.Separator)) && absTemplate != absConfig {
				return nil, fmt.Errorf("template path escapes config directory: %s", templatePath)
			}
		}

		data, err = applyTemplate(data, absTemplatePath)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// parseRefToken parses "path?filter=x:y&template=z" into components
func parseRefToken(token string) (path string, filters map[string]string, templatePath string, err error) {
	filters = make(map[string]string)

	// Split path and query string
	parts := strings.SplitN(token, "?", 2)
	path = parts[0]

	if len(parts) == 1 {
		return path, filters, "", nil
	}

	// Parse query params
	query, err := url.ParseQuery(parts[1])
	if err != nil {
		return "", nil, "", fmt.Errorf("invalid query string: %w", err)
	}

	// Extract filters (filter=field:value)
	for _, f := range query["filter"] {
		idx := strings.Index(f, ":")
		if idx == -1 {
			return "", nil, "", fmt.Errorf("invalid filter format %q, expected field:value", f)
		}
		field := f[:idx]
		value := f[idx+1:]

		// Check for duplicate filter on same field
		if _, exists := filters[field]; exists {
			return "", nil, "", fmt.Errorf("duplicate filter for field %q", field)
		}
		filters[field] = value
	}

	// Extract template path
	if t := query.Get("template"); t != "" {
		templatePath = t
	}

	return path, filters, templatePath, nil
}

// loadRefData loads data from a file or directory
func loadRefData(absPath string, directoryIntent bool) (interface{}, error) {
	// Check if it's a directory (explicit intent or is a directory on disk)
	isDir := directoryIntent
	if !isDir {
		info, err := os.Stat(absPath)
		if err == nil && info.IsDir() {
			isDir = true
		}
	}

	if isDir {
		// Directory: use persist.ReadDir to aggregate JSON files
		dirPath := strings.TrimSuffix(absPath, "/")
		data, err := persist.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		var result []interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parsing directory data: %w", err)
		}
		return result, nil
	}

	// Single file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", absPath, err)
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing file %q: %w", absPath, err)
	}
	return result, nil
}

// filterData filters data by field:value equality
func filterData(data interface{}, filters map[string]string) (interface{}, error) {
	// If data is an array, filter each item
	if arr, ok := data.([]interface{}); ok {
		var result []interface{}
		for _, item := range arr {
			if matchesFilters(item, filters) {
				result = append(result, item)
			}
		}
		if result == nil {
			result = []interface{}{} // Return empty array, not null
		}
		return result, nil
	}

	// If data is an object, return it if matches, empty array otherwise
	if matchesFilters(data, filters) {
		return data, nil
	}
	return []interface{}{}, nil
}

// matchesFilters checks if an item matches all filters
func matchesFilters(item interface{}, filters map[string]string) bool {
	obj, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range filters {
		actual := extractNestedValue(obj, field)
		if fmt.Sprintf("%v", actual) != expected {
			return false
		}
	}
	return true
}

// extractNestedValue extracts a value using dot notation (e.g., "user.role")
func extractNestedValue(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

// applyTemplate transforms data using a Go template file
func applyTemplate(data interface{}, templatePath string) (interface{}, error) {
	// Read template file
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("reading template %q: %w", templatePath, err)
	}

	tmpl, err := template.New("ref").Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template %q: %w", templatePath, err)
	}

	// Apply template based on data type
	switch v := data.(type) {
	case []interface{}:
		var result []interface{}
		for i, item := range v {
			transformed, err := executeTemplate(tmpl, item)
			if err != nil {
				return nil, fmt.Errorf("applying template to item %d: %w", i, err)
			}
			result = append(result, transformed)
		}
		if result == nil {
			result = []interface{}{}
		}
		return result, nil

	default:
		return executeTemplate(tmpl, data)
	}
}

// executeTemplate executes a template against data and returns parsed JSON
func executeTemplate(tmpl *template.Template, data interface{}) (interface{}, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing template output: %w", err)
	}
	return result, nil
}
