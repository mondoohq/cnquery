// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/stretchr/testify/assert"
)

func TestServerImage(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		assert.Equal(t, "", serverImage(nil))
	})
	t.Run("non-map returns empty", func(t *testing.T) {
		assert.Equal(t, "", serverImage(""))
		assert.Equal(t, "", serverImage("some-string"))
		assert.Equal(t, "", serverImage(123))
	})
	t.Run("map with id returns id", func(t *testing.T) {
		assert.Equal(t, "img-1", serverImage(map[string]any{"id": "img-1"}))
	})
	t.Run("map without id returns empty", func(t *testing.T) {
		assert.Equal(t, "", serverImage(map[string]any{"name": "n"}))
	})
	t.Run("map with non-string id returns empty", func(t *testing.T) {
		assert.Equal(t, "", serverImage(map[string]any{"id": 123}))
	})
}

func TestServerFlavorRef(t *testing.T) {
	t.Run("nil returns empty pair", func(t *testing.T) {
		id, name := serverFlavorRef(nil)
		assert.Equal(t, "", id)
		assert.Equal(t, "", name)
	})
	t.Run("microversion <2.47: id only", func(t *testing.T) {
		id, name := serverFlavorRef(map[string]any{"id": "uuid-1"})
		assert.Equal(t, "uuid-1", id)
		assert.Equal(t, "", name)
	})
	t.Run("microversion >=2.47: original_name only", func(t *testing.T) {
		id, name := serverFlavorRef(map[string]any{"original_name": "m1.small"})
		assert.Equal(t, "", id)
		assert.Equal(t, "m1.small", name)
	})
	t.Run("both fields present", func(t *testing.T) {
		id, name := serverFlavorRef(map[string]any{"id": "uuid-1", "original_name": "m1.small"})
		assert.Equal(t, "uuid-1", id)
		assert.Equal(t, "m1.small", name)
	})
	t.Run("non-string values are ignored", func(t *testing.T) {
		id, name := serverFlavorRef(map[string]any{"id": 1, "original_name": true})
		assert.Equal(t, "", id)
		assert.Equal(t, "", name)
	})
}

func TestServerSecurityGroupNames(t *testing.T) {
	t.Run("nil and empty return empty slice", func(t *testing.T) {
		assert.Empty(t, serverSecurityGroupNames(nil))
		assert.Empty(t, serverSecurityGroupNames([]map[string]any{}))
	})
	t.Run("extracts names, skipping empty and non-string", func(t *testing.T) {
		got := serverSecurityGroupNames([]map[string]any{
			{"name": "default"},
			{"name": ""},
			{"name": 42},
			{"id": "no-name"},
			{"name": "web"},
		})
		assert.Equal(t, []string{"default", "web"}, got)
	})
}

func TestServerVolumeIDs(t *testing.T) {
	t.Run("nil and empty return empty slice", func(t *testing.T) {
		assert.Empty(t, serverVolumeIDs(nil))
	})
	t.Run("extracts non-empty IDs", func(t *testing.T) {
		got := serverVolumeIDs([]servers.AttachedVolume{
			{ID: "v-1"},
			{ID: ""},
			{ID: "v-2"},
		})
		assert.Equal(t, []string{"v-1", "v-2"}, got)
	})
}

func TestDerefStrings(t *testing.T) {
	t.Run("nil pointer returns nil", func(t *testing.T) {
		assert.Nil(t, derefStrings(nil))
	})
	t.Run("non-nil pointer returns slice", func(t *testing.T) {
		s := []string{"a", "b"}
		assert.Equal(t, s, derefStrings(&s))
	})
	t.Run("pointer to empty slice returns empty", func(t *testing.T) {
		s := []string{}
		assert.Equal(t, s, derefStrings(&s))
	})
}

func TestToDict(t *testing.T) {
	t.Run("nil returns non-nil empty map", func(t *testing.T) {
		got := toDict(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("non-nil passes through", func(t *testing.T) {
		in := map[string]any{"k": "v"}
		assert.Equal(t, in, toDict(in))
	})
}

func TestServerGroupMetadata(t *testing.T) {
	t.Run("nil and empty inputs return nil", func(t *testing.T) {
		assert.Nil(t, serverGroupMetadata(nil))
		assert.Nil(t, serverGroupMetadata(map[string]any{}))
	})

	t.Run("strings pass through unchanged", func(t *testing.T) {
		got := serverGroupMetadata(map[string]any{
			"a": "alpha",
			"b": "",
		})
		assert.Equal(t, map[string]string{"a": "alpha", "b": ""}, got)
	})

	t.Run("nil values are skipped", func(t *testing.T) {
		got := serverGroupMetadata(map[string]any{
			"present": "x",
			"absent":  nil,
		})
		assert.Equal(t, map[string]string{"present": "x"}, got)
	})

	t.Run("non-string scalars are JSON-encoded", func(t *testing.T) {
		got := serverGroupMetadata(map[string]any{
			"flag":    true,
			"count":   float64(123),
			"missing": false,
		})
		assert.Equal(t, map[string]string{
			"flag":    "true",
			"count":   "123",
			"missing": "false",
		}, got)
	})

	t.Run("collections render as JSON", func(t *testing.T) {
		got := serverGroupMetadata(map[string]any{
			"tags":    []any{"a", "b"},
			"nested":  map[string]any{"k": "v"},
			"numbers": []any{float64(1), float64(2)},
		})
		assert.Equal(t, `["a","b"]`, got["tags"])
		assert.Equal(t, `{"k":"v"}`, got["nested"])
		assert.Equal(t, `[1,2]`, got["numbers"])
	})
}
