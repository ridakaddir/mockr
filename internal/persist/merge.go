package persist

// DeepMerge merges two maps recursively. Values from overlay take precedence
// over base. Nested maps (map[string]interface{}) are merged recursively;
// all other types are overwritten by the overlay value.
//
// Neither base nor overlay is mutated — the result is a new map.
func DeepMerge(base, overlay map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base)+len(overlay))

	for k, v := range base {
		result[k] = v
	}

	for k, v := range overlay {
		if baseMap, ok := result[k].(map[string]interface{}); ok {
			if overlayMap, ok := v.(map[string]interface{}); ok {
				result[k] = DeepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}
