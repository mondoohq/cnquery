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

func TestClassifyPatch_StrategicMerge(t *testing.T) {
	// A mapping with apiVersion/kind is a strategic-merge patch.
	raw := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
`)
	format, ops := classifyPatch(raw, hintNone)
	assert.Equal(t, patchFormatStrategicMerge, format)
	assert.Empty(t, ops, "strategic-merge patches carry no decomposed operations")
}

func TestClassifyPatch_JSON6902(t *testing.T) {
	// A sequence whose elements each carry an `op` key is JSON6902.
	raw := []byte(`- op: add
  path: /spec/template/spec/containers/0/env
  value:
    - name: LOG_LEVEL
      value: debug
- op: remove
  path: /spec/template/spec/containers/0/securityContext
`)
	format, ops := classifyPatch(raw, hintNone)
	assert.Equal(t, patchFormatJSON6902, format)
	assert.Len(t, ops, 2)

	assert.Equal(t, "add", ops[0].op)
	assert.Equal(t, "/spec/template/spec/containers/0/env", ops[0].path)
	assert.True(t, ops[0].hasValue, "add carries a value")
	assert.NotNil(t, ops[0].value)

	assert.Equal(t, "remove", ops[1].op)
	assert.Equal(t, "/spec/template/spec/containers/0/securityContext", ops[1].path)
	assert.False(t, ops[1].hasValue, "remove carries no value")
}

func TestClassifyPatch_JSON6902InlineArray(t *testing.T) {
	// JSON array syntax is equally valid and classifies the same way.
	raw := []byte(`[{"op":"replace","path":"/spec/replicas","value":5}]`)
	format, ops := classifyPatch(raw, hintNone)
	assert.Equal(t, patchFormatJSON6902, format)
	assert.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].op)
	assert.Equal(t, "/spec/replicas", ops[0].path)
	assert.True(t, ops[0].hasValue)
	assert.EqualValues(t, 5, ops[0].value)
}

func TestClassifyPatch_SequenceWithoutOpIsStrategicMerge(t *testing.T) {
	// A sequence whose elements lack `op` is not a JSON6902 patch.
	raw := []byte(`- name: a
  value: 1
`)
	format, ops := classifyPatch(raw, hintNone)
	assert.Equal(t, patchFormatStrategicMerge, format)
	assert.Empty(t, ops)
}

func TestClassifyPatch_Hints(t *testing.T) {
	// A forced strategic-merge hint never decomposes operations even if the
	// content looks like JSON6902.
	json6902 := []byte(`- op: remove
  path: /spec
`)
	format, ops := classifyPatch(json6902, hintStrategicMerge)
	assert.Equal(t, patchFormatStrategicMerge, format)
	assert.Empty(t, ops)

	// A forced json6902 hint decodes whatever operations are present.
	format, ops = classifyPatch(json6902, hintJSON6902)
	assert.Equal(t, patchFormatJSON6902, format)
	assert.Len(t, ops, 1)
	assert.Equal(t, "remove", ops[0].op)
}

func TestClassifyPatch_MalformedIsStrategicMerge(t *testing.T) {
	// Malformed YAML must never panic and falls back to strategic-merge.
	format, ops := classifyPatch([]byte("this: : : not valid"), hintNone)
	assert.Equal(t, patchFormatStrategicMerge, format)
	assert.Empty(t, ops)

	// Empty content is strategic-merge with no operations.
	format, ops = classifyPatch(nil, hintNone)
	assert.Equal(t, patchFormatStrategicMerge, format)
	assert.Empty(t, ops)
}
