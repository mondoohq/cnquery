// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenNspProperties(t *testing.T) {
	t.Run("lifts nested properties to the top level", func(t *testing.T) {
		in := map[string]any{
			"name": "rule-1",
			"properties": map[string]any{
				"direction":       "Inbound",
				"addressPrefixes": []any{"10.0.0.0/8"},
			},
		}
		out := flattenNspProperties(in)
		m, ok := out.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "rule-1", m["name"])
		assert.Equal(t, "Inbound", m["direction"])
		assert.Equal(t, []any{"10.0.0.0/8"}, m["addressPrefixes"])
		// the envelope key is gone
		_, hasProps := m["properties"]
		assert.False(t, hasProps)
	})

	t.Run("returns the dict unchanged when there is no properties envelope", func(t *testing.T) {
		in := map[string]any{"name": "rule-1", "direction": "Outbound"}
		out := flattenNspProperties(in)
		assert.Equal(t, in, out)
	})

	t.Run("returns non-map input unchanged", func(t *testing.T) {
		assert.Equal(t, "not-a-map", flattenNspProperties("not-a-map"))
		assert.Nil(t, flattenNspProperties(nil))
	})

	t.Run("ignores a properties value that is not a map", func(t *testing.T) {
		in := map[string]any{"name": "rule-1", "properties": "oops"}
		out := flattenNspProperties(in)
		assert.Equal(t, in, out)
	})
}
