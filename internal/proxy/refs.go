package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ridakaddir/mockr/internal/persist"
)

// refPattern matches "{{ref:path?params}}" tokens (including surrounding quotes)
var refPattern = regexp.MustCompile(`"{{ref:([^}]*(?:\{[^}]+\}[^}]*)*)}}"`)

// dynamicPlaceholderPattern matches placeholders inside ref tokens: {.field}, {path.x}, {query.x}, {header.x}
var dynamicPlaceholderPattern = regexp.MustCompile(`\{(\.[\w.]+|path\.[\w]+|query\.[\w]+|header\.[\w-]+)\}`)

// RefContext holds the context needed to resolve dynamic placeholders in refs
type RefContext struct {
	Body       map[string]interface{} // Request body fields
	PathParams map[string]string      // URL path parameters
	Query      url.Values             // Query parameters
	Headers    http.Header            // Request headers
}

// sanitizeHeaders returns a copy of the given headers with sensitive entries removed.
func sanitizeHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	safe := make(http.Header, len(h))
	for k, values := range h {
		switch strings.ToLower(k) {
		case "authorization", "cookie", "proxy-authorization":
			// Skip sensitive headers
			continue
		default:
			// Copy slice to avoid sharing underlying array
			copied := make([]string, len(values))
			copy(copied, values)
			safe[k] = copied
		}
	}
	return safe
}

// NewRefContext creates a RefContext from an HTTP request
func NewRefContext(r *http.Request, bodyBytes []byte, pathParams map[string]string) *RefContext {
	ctx := &RefContext{
		PathParams: pathParams,
		Query:      r.URL.Query(),
		Headers:    sanitizeHeaders(r.Header),
	}

	// Parse body JSON if present
	if len(bodyBytes) > 0 {
		var body map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &body); err == nil {
			ctx.Body = body
		}
	}
	if ctx.Body == nil {
		ctx.Body = make(map[string]interface{})
	}

	return ctx
}

// resolveRefsWithContext combines dynamic placeholder resolution + ref resolution.
// NOTE: Dynamic placeholders are resolved only in the initial/top-level content passed
// to this function. Any {{ref:...}} tokens found in files loaded during recursive
// resolution are processed by resolveRefs without a RefContext, so placeholders inside
// those nested refs are treated literally.
func resolveRefsWithContext(content []byte, configDir string, visited map[string]bool, ctx *RefContext) ([]byte, error) {
	// Step 1: Resolve spread references first (they need to operate on the original JSON structure)
	content, err := resolveSpreadRefs(content, configDir, visited, ctx)
	if err != nil {
		return nil, err
	}

	// Step 2: Resolve dynamic placeholders in ref tokens in the remaining content
	content, err = resolveDynamicInRefs(content, ctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Resolve the refs themselves (including any nested refs)
	return resolveRefs(content, configDir, visited)
}

// resolveDynamicInRefs resolves {.field}, {path.x}, {query.x}, {header.x} placeholders
// inside {{ref:...}} tokens in a single blob of content. It is intended to be called
// on the top-level content before recursive ref resolution begins.
func resolveDynamicInRefs(content []byte, ctx *RefContext) ([]byte, error) {
	// Find all {{ref:...}} tokens
	matches := refPattern.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Check if any ref tokens contain placeholders
	if ctx == nil {
		for _, match := range matches {
			refToken := string(content[match[2]:match[3]])
			placeholders := dynamicPlaceholderPattern.FindAllStringSubmatch(refToken, -1)
			if len(placeholders) > 0 {
				return nil, fmt.Errorf("dynamic placeholders found in ref token %q but no request context available", refToken)
			}
		}
		return content, nil
	}

	result := make([]byte, len(content))
	copy(result, content)

	// Process in reverse order to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		refToken := string(content[match[2]:match[3]]) // The path?params part

		// Resolve placeholders in this ref token
		resolved, err := resolvePlaceholders(refToken, ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving placeholders in ref %q: %w", refToken, err)
		}

		// Build new token
		newToken := fmt.Sprintf(`"{{ref:%s}}"`, resolved)

		// Replace in result using safe buffer approach
		prefix := result[:match[0]]
		suffix := result[match[1]:]
		buf := make([]byte, 0, len(prefix)+len(newToken)+len(suffix))
		buf = append(buf, prefix...)
		buf = append(buf, newToken...)
		buf = append(buf, suffix...)
		result = buf
	}

	return result, nil
}

// resolvePlaceholders resolves all {placeholder} patterns in a ref token.
// The caller must ensure ctx is not nil.
func resolvePlaceholders(token string, ctx *RefContext) (string, error) {
	result := token

	// Find all placeholders
	placeholders := dynamicPlaceholderPattern.FindAllStringSubmatch(token, -1)

	for _, match := range placeholders {
		placeholder := match[0] // e.g., {.endpointId}
		key := match[1]         // e.g., .endpointId

		value, err := resolveValue(key, ctx)
		if err != nil {
			return "", fmt.Errorf("cannot resolve %s: %w", placeholder, err)
		}

		// Sanitize the value to prevent injection of ref query parameters
		safeValue, err := sanitizeRefPlaceholderValue(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for %s: %w", placeholder, err)
		}

		result = strings.Replace(result, placeholder, safeValue, 1)
	}

	return result, nil
}

// sanitizeRefPlaceholderValue ensures that a placeholder value cannot inject or
// override ref query parameters by containing reserved characters such as
// '?', '&', or '='. Such values are rejected.
func sanitizeRefPlaceholderValue(value string) (string, error) {
	if strings.ContainsAny(value, "?&=") {
		return "", fmt.Errorf("placeholder value contains reserved characters (?&=)")
	}
	return value, nil
}

// resolveValue resolves a single placeholder key to its value
func resolveValue(key string, ctx *RefContext) (string, error) {
	switch {
	case strings.HasPrefix(key, "."):
		// Body field: {.endpointId} or {.nested.field}
		fieldPath := key[1:] // Remove leading dot
		value := extractNestedValue(ctx.Body, fieldPath)
		if value == nil {
			return "", fmt.Errorf("field %q not found in request body", fieldPath)
		}
		strValue := fmt.Sprintf("%v", value)
		if strValue == "" {
			return "", fmt.Errorf("field %q resolved to empty string", fieldPath)
		}
		return strValue, nil

	case strings.HasPrefix(key, "path."):
		// Path param: {path.endpointId}
		paramName := key[5:]
		value, ok := ctx.PathParams[paramName]
		if !ok {
			return "", fmt.Errorf("path parameter %q not found", paramName)
		}
		if value == "" {
			return "", fmt.Errorf("path parameter %q is empty", paramName)
		}
		return value, nil

	case strings.HasPrefix(key, "query."):
		// Query param: {query.version}
		paramName := key[6:]
		value := ctx.Query.Get(paramName)
		if value == "" {
			return "", fmt.Errorf("query parameter %q not found or empty", paramName)
		}
		return value, nil

	case strings.HasPrefix(key, "header."):
		// Header: {header.X-Tenant-Id}
		headerName := key[7:]
		value := ctx.Headers.Get(headerName)
		if value == "" {
			return "", fmt.Errorf("header %q not found or empty", headerName)
		}
		return value, nil

	default:
		return "", fmt.Errorf("unknown placeholder type: %s", key)
	}
}

// resolveFileRefs processes {{ref:...}} tokens in JSON content, but only resolves
// refs that point to single files (not directories). Directory refs (paths ending
// with /) are preserved as raw tokens so they remain live references that resolve
// dynamically on each read. This is used by loadDefaults to prevent baking
// directory aggregation results (e.g. deployment lists) into persisted files.
func resolveFileRefs(content []byte, configDir string, visited map[string]bool) ([]byte, error) {
	if len(content) == 0 || len(strings.TrimSpace(string(content))) == 0 {
		return content, nil
	}

	matches := refPattern.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Process matches in reverse order to preserve indices.
	result := content
	changed := false
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		token := string(content[match[2]:match[3]]) // path?params

		// Extract the path portion (before any ?query params).
		refPath := token
		if idx := strings.Index(refPath, "?"); idx != -1 {
			refPath = refPath[:idx]
		}

		// Skip directory refs — preserve them as live references.
		if strings.HasSuffix(refPath, "/") {
			continue
		}

		// Resolve file-based refs normally.
		resolved, err := resolveRefToken(token, configDir, visited)
		if err != nil {
			return nil, fmt.Errorf("resolving ref %q: %w", token, err)
		}

		jsonBytes, err := json.Marshal(resolved)
		if err != nil {
			return nil, fmt.Errorf("marshalling ref result: %w", err)
		}

		prefix := result[:match[0]]
		suffix := result[match[1]:]
		buf := make([]byte, 0, len(prefix)+len(jsonBytes)+len(suffix))
		buf = append(buf, prefix...)
		buf = append(buf, jsonBytes...)
		buf = append(buf, suffix...)
		result = buf
		changed = true
	}

	// Validate JSON only when we actually made replacements.
	if changed {
		var validation interface{}
		if err := json.Unmarshal(result, &validation); err != nil {
			return nil, fmt.Errorf("invalid JSON after ref resolution: %w", err)
		}
	}

	return result, nil
}

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

// resolveSpreadRefs processes all $spread field references in content
// and spreads the referenced object properties into the containing object
func resolveSpreadRefs(content []byte, configDir string, visited map[string]bool, refCtx *RefContext) ([]byte, error) {
	// Skip processing if content is empty or whitespace-only
	if len(content) == 0 || len(strings.TrimSpace(string(content))) == 0 {
		return content, nil
	}

	// Quick check: if content doesn't contain "$spread", return unchanged
	if !bytes.Contains(content, []byte("$spread")) {
		return content, nil
	}

	// Parse the JSON to find $spread fields
	var parsed interface{}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, fmt.Errorf("parsing JSON for spread resolution: %w", err)
	}

	// Process spread fields recursively
	result, err := processSpreadFields(parsed, configDir, visited, refCtx)
	if err != nil {
		return nil, err
	}

	// Marshal the result back to JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshalling spread result: %w", err)
	}

	return resultBytes, nil
}

// processSpreadFields recursively finds and processes $spread fields in the parsed JSON structure
func processSpreadFields(data interface{}, configDir string, visited map[string]bool, refCtx *RefContext) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this object contains a $spread field
		if spreadRef, exists := v["$spread"]; exists {
			// Process the spread reference
			spreadRefStr, ok := spreadRef.(string)
			if !ok {
				return nil, fmt.Errorf("$spread field must be a string, got %T", spreadRef)
			}

			// Resolve the spread reference to get object properties
			spreadObj, err := processSpreadRef(spreadRefStr, configDir, visited, refCtx)
			if err != nil {
				return nil, fmt.Errorf("resolving $spread reference %q: %w", spreadRefStr, err)
			}

			// Create new object with spread properties first, then existing properties
			newObj := make(map[string]interface{})

			// Add spread properties first
			for key, value := range spreadObj {
				newObj[key] = value
			}

			// Add existing properties (these will override spread properties if there are conflicts)
			// Skip the $spread field itself
			for key, value := range v {
				if key == "$spread" {
					continue
				}
				// Recursively process nested structures
				processed, err := processSpreadFields(value, configDir, visited, refCtx)
				if err != nil {
					return nil, err
				}
				newObj[key] = processed
			}

			return newObj, nil
		}

		// No $spread field, process nested objects recursively
		newObj := make(map[string]interface{})
		for key, value := range v {
			processed, err := processSpreadFields(value, configDir, visited, refCtx)
			if err != nil {
				return nil, err
			}
			newObj[key] = processed
		}
		return newObj, nil

	case []interface{}:
		// Process arrays recursively
		newArray := make([]interface{}, len(v))
		for i, item := range v {
			processed, err := processSpreadFields(item, configDir, visited, refCtx)
			if err != nil {
				return nil, err
			}
			newArray[i] = processed
		}
		return newArray, nil

	default:
		// Return primitive values as-is
		return data, nil
	}
}

// processSpreadRef resolves a single spread reference and returns the object to be spread
func processSpreadRef(refToken string, configDir string, visited map[string]bool, refCtx *RefContext) (map[string]interface{}, error) {
	// Check if this is a {{ref:...}} token and extract the path
	if !strings.HasPrefix(refToken, "{{ref:") || !strings.HasSuffix(refToken, "}}") {
		return nil, fmt.Errorf("$spread value must be a {{ref:...}} token, got %q", refToken)
	}

	// Extract the path from {{ref:path}}
	path := refToken[6 : len(refToken)-2] // Remove "{{ref:" and "}}"

	// First resolve any dynamic placeholders in the ref token
	if refCtx != nil {
		resolved, err := resolvePlaceholders(path, refCtx)
		if err != nil {
			return nil, fmt.Errorf("resolving placeholders in spread ref %q: %w", path, err)
		}
		path = resolved
	}

	// Resolve the reference using existing logic
	resolved, err := resolveRefToken(path, configDir, visited)
	if err != nil {
		return nil, err
	}

	// Ensure the result is an object that can be spread
	obj, ok := resolved.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$spread ref must resolve to an object, got %T", resolved)
	}

	return obj, nil
}
