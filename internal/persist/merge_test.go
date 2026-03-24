package persist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepMergeDisjointKeys(t *testing.T) {
	base := map[string]interface{}{
		"a": "from-base",
		"b": 42,
	}
	overlay := map[string]interface{}{
		"c": "from-overlay",
		"d": true,
	}

	result := DeepMerge(base, overlay)

	assert.Equal(t, "from-base", result["a"])
	assert.Equal(t, 42, result["b"])
	assert.Equal(t, "from-overlay", result["c"])
	assert.Equal(t, true, result["d"])
	assert.Len(t, result, 4)
}

func TestDeepMergeOverlayWins(t *testing.T) {
	base := map[string]interface{}{
		"name":   "default-name",
		"status": "pending",
	}
	overlay := map[string]interface{}{
		"name": "user-provided",
	}

	result := DeepMerge(base, overlay)

	assert.Equal(t, "user-provided", result["name"])
	assert.Equal(t, "pending", result["status"])
}

func TestDeepMergeRecursive(t *testing.T) {
	base := map[string]interface{}{
		"spec": map[string]interface{}{
			"machineType":  "e2-standard-2",
			"replicaCount": 1,
		},
	}
	overlay := map[string]interface{}{
		"spec": map[string]interface{}{
			"machineType": "n1-standard-4",
		},
	}

	result := DeepMerge(base, overlay)

	spec := result["spec"].(map[string]interface{})
	assert.Equal(t, "n1-standard-4", spec["machineType"])
	assert.Equal(t, 1, spec["replicaCount"]) // preserved from base
}

func TestDeepMergeOverlayReplacesNested(t *testing.T) {
	base := map[string]interface{}{
		"data": map[string]interface{}{
			"key": "value",
		},
	}
	overlay := map[string]interface{}{
		"data": "scalar-value",
	}

	result := DeepMerge(base, overlay)

	assert.Equal(t, "scalar-value", result["data"])
}

func TestDeepMergeOverlayMapOverScalar(t *testing.T) {
	base := map[string]interface{}{
		"data": "scalar-value",
	}
	overlay := map[string]interface{}{
		"data": map[string]interface{}{
			"key": "value",
		},
	}

	result := DeepMerge(base, overlay)

	dataMap := result["data"].(map[string]interface{})
	assert.Equal(t, "value", dataMap["key"])
}

func TestDeepMergeEmptyOverlay(t *testing.T) {
	base := map[string]interface{}{
		"a": "value",
		"b": 42,
	}

	result := DeepMerge(base, map[string]interface{}{})

	assert.Equal(t, base, result)
	// Verify it's a copy, not the same map
	base["a"] = "mutated"
	assert.Equal(t, "value", result["a"])
}

func TestDeepMergeEmptyBase(t *testing.T) {
	overlay := map[string]interface{}{
		"a": "value",
		"b": 42,
	}

	result := DeepMerge(map[string]interface{}{}, overlay)

	assert.Equal(t, overlay, result)
	// Verify it's a copy, not the same map
	overlay["a"] = "mutated"
	assert.Equal(t, "value", result["a"])
}

func TestDeepMergeBothEmpty(t *testing.T) {
	result := DeepMerge(map[string]interface{}{}, map[string]interface{}{})

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestDeepMergeNilValues(t *testing.T) {
	base := map[string]interface{}{
		"a": "value",
		"b": 42,
	}
	overlay := map[string]interface{}{
		"a": nil,
	}

	result := DeepMerge(base, overlay)

	assert.Nil(t, result["a"])
	assert.Equal(t, 42, result["b"])
}

func TestDeepMergeDeeplyNested(t *testing.T) {
	base := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"fromBase":  "base-value",
					"sharedKey": "base-version",
				},
			},
			"otherKey": "preserved",
		},
	}
	overlay := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"sharedKey":   "overlay-version",
					"fromOverlay": "overlay-value",
				},
			},
		},
	}

	result := DeepMerge(base, overlay)

	l1 := result["level1"].(map[string]interface{})
	assert.Equal(t, "preserved", l1["otherKey"])

	l2 := l1["level2"].(map[string]interface{})
	l3 := l2["level3"].(map[string]interface{})
	assert.Equal(t, "base-value", l3["fromBase"])
	assert.Equal(t, "overlay-version", l3["sharedKey"])
	assert.Equal(t, "overlay-value", l3["fromOverlay"])
}

func TestDeepMergeDoesNotMutateInputs(t *testing.T) {
	base := map[string]interface{}{
		"nested": map[string]interface{}{
			"a": "original",
		},
	}
	overlay := map[string]interface{}{
		"nested": map[string]interface{}{
			"b": "added",
		},
	}

	result := DeepMerge(base, overlay)

	// Mutate result and verify originals are unaffected
	resultNested := result["nested"].(map[string]interface{})
	resultNested["c"] = "mutation"

	baseNested := base["nested"].(map[string]interface{})
	_, hasC := baseNested["c"]
	assert.False(t, hasC, "base should not be mutated")

	overlayNested := overlay["nested"].(map[string]interface{})
	_, hasC = overlayNested["c"]
	assert.False(t, hasC, "overlay should not be mutated")
}
