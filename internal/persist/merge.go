package persist

// DeepMerge merges two maps recursively. Values from overlay take precedence
// over base. Nested maps (map[string]interface{}) are merged recursively;
// all other types are overwritten by the overlay value.
//
// Neither base nor overlay is mutated — the result is a fully independent map.
// Nested map values are deep-copied even when no key conflict triggers a
// recursive merge, so callers can freely mutate the result without affecting
// the original inputs.
func DeepMerge(base, overlay map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base)+len(overlay))

	for k, v := range base {
		result[k] = deepCopyValue(v)
	}

	for k, v := range overlay {
		if baseMap, ok := result[k].(map[string]interface{}); ok {
			if overlayMap, ok := v.(map[string]interface{}); ok {
				result[k] = DeepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = deepCopyValue(v)
	}

	return result
}

// deepCopyValue returns a deep copy of v if it is a map[string]interface{},
// otherwise returns v as-is (scalar values, slices, etc. are shared).
func deepCopyValue(v interface{}) interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		cp := make(map[string]interface{}, len(m))
		for k, val := range m {
			cp[k] = deepCopyValue(val)
		}
		return cp
	}
	return v
}
