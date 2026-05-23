// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Kubernetes requires label/annotation values to be strings, but YAML
// unmarshaling can hand us ints, bools, or nil when a manifest is
// non-compliant. coerceStringMap preserves every value as a string
// rather than silently dropping the non-string ones.
func TestCoerceStringMap(t *testing.T) {
	in := map[string]any{
		"app":      "demo",
		"replicas": 3,                // int from `replicas: 3`
		"enabled":  true,             // bool
		"none":     nil,              // nil
		"f":        3.14,             // float
		"nested":   map[string]int{}, // arbitrary, stringified
	}
	got := coerceStringMap(in)

	assert.Equal(t, "demo", got["app"])
	assert.Equal(t, "3", got["replicas"], "ints survive as decimal strings")
	assert.Equal(t, "true", got["enabled"], "bools survive as 'true'/'false'")
	assert.Equal(t, "", got["none"], "nil becomes empty string")
	assert.Equal(t, "3.14", got["f"])
	assert.Contains(t, got["nested"], "map", "fallback %v rendering for unknowns")
	assert.Len(t, got, len(in), "no entries dropped")
}

func TestCoerceStringMap_EmptyInputReturnsNonNil(t *testing.T) {
	got := coerceStringMap(nil)
	assert.NotNil(t, got, "downstream MapData expects a non-nil map")
	assert.Len(t, got, 0)
}

func TestCoerceStringMap_EmptyMap(t *testing.T) {
	got := coerceStringMap(map[string]any{})
	assert.NotNil(t, got)
	assert.Len(t, got, 0)
}
