package generate

import (
	"encoding/json"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// marshalExample encodes an arbitrary value (from spec examples) as indented JSON.
// Returns nil if encoding fails or value is nil.
func marshalExample(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil
	}
	return b
}

// SynthesiseExample generates a minimal valid Go value from an OpenAPI schema.
// maxDepth prevents infinite recursion on deeply nested or circular schemas.
func SynthesiseExample(schema *openapi3.Schema, maxDepth int) any {
	if schema == nil || maxDepth <= 0 {
		return nil
	}

	// Use spec-provided example if available.
	if schema.Example != nil {
		return schema.Example
	}

	// Use first enum value if present.
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	// Use default value if present.
	if schema.Default != nil {
		return schema.Default
	}

	// Handle composition keywords by picking / merging sub-schemas.
	if len(schema.OneOf) > 0 && schema.OneOf[0] != nil && schema.OneOf[0].Value != nil {
		return SynthesiseExample(schema.OneOf[0].Value, maxDepth-1)
	}
	if len(schema.AnyOf) > 0 && schema.AnyOf[0] != nil && schema.AnyOf[0].Value != nil {
		return SynthesiseExample(schema.AnyOf[0].Value, maxDepth-1)
	}
	if len(schema.AllOf) > 0 {
		merged := make(map[string]any)
		for _, ref := range schema.AllOf {
			if ref == nil || ref.Value == nil {
				continue
			}
			if sub, ok := SynthesiseExample(ref.Value, maxDepth-1).(map[string]any); ok {
				for k, v := range sub {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			return merged
		}
	}

	// Use Types.Is for type detection (schema.Type is *openapi3.Types, a []string pointer).
	t := schema.Type

	if t.Is(openapi3.TypeObject) || (t == nil && len(schema.Properties) > 0) {
		return synthObject(schema, maxDepth)
	}
	if t.Is(openapi3.TypeArray) {
		return synthArray(schema, maxDepth)
	}
	if t.Is(openapi3.TypeString) {
		return synthString(schema)
	}
	if t.Is(openapi3.TypeInteger) {
		return synthInt(schema)
	}
	if t.Is(openapi3.TypeNumber) {
		return synthFloat(schema)
	}
	if t.Is(openapi3.TypeBoolean) {
		return true
	}
	if t.Is(openapi3.TypeNull) {
		return nil
	}

	// No type declared — fall back to object if properties exist.
	if len(schema.Properties) > 0 {
		return synthObject(schema, maxDepth)
	}
	return nil
}

func synthObject(schema *openapi3.Schema, depth int) map[string]any {
	obj := make(map[string]any)
	if schema.Properties == nil {
		return obj
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(schema.Properties))
	for k := range schema.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		ref := schema.Properties[name]
		if ref == nil || ref.Value == nil {
			obj[name] = nil
			continue
		}
		obj[name] = SynthesiseExample(ref.Value, depth-1)
	}
	return obj
}

func synthArray(schema *openapi3.Schema, depth int) []any {
	if schema.Items == nil || schema.Items.Value == nil {
		return []any{}
	}
	item := SynthesiseExample(schema.Items.Value, depth-1)
	return []any{item}
}

func synthString(schema *openapi3.Schema) string {
	switch schema.Format {
	case "uuid":
		return "{{uuid}}"
	case "date-time":
		return "{{now}}"
	case "date":
		return "2026-01-01"
	case "email":
		return "user@example.com"
	case "uri", "url":
		return "https://example.com"
	case "hostname":
		return "example.com"
	case "ipv4":
		return "127.0.0.1"
	case "ipv6":
		return "::1"
	case "byte":
		return "c3RyaW5n" // base64("string")
	case "password":
		return "********"
	}
	if schema.Title != "" {
		return schema.Title
	}
	return "string"
}

func synthInt(schema *openapi3.Schema) int64 {
	if schema.Min != nil && *schema.Min > 0 {
		return int64(*schema.Min)
	}
	return 1
}

func synthFloat(schema *openapi3.Schema) float64 {
	if schema.Min != nil && *schema.Min > 0 {
		return *schema.Min
	}
	return 1.0
}

// SynthesiseJSON generates indented JSON bytes from a schema.
// Returns []byte(`{}`) if synthesis fails.
func SynthesiseJSON(schema *openapi3.Schema) []byte {
	if schema == nil {
		return []byte(`{}`)
	}
	val := SynthesiseExample(schema, 5)
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
