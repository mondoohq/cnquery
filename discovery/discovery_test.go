// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestDiscoveredAssetsAdd(t *testing.T) {
	t.Run("exact duplicate is rejected", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a1 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}
		a2 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}

		assert.True(t, d.Add(a1, nil))
		assert.False(t, d.Add(a2, nil))
		assert.Len(t, d.Assets, 1)
	})

	t.Run("shared ssh host key with different hostname is not a duplicate", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a1 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}
		a2 := &inventory.Asset{PlatformIds: []string{"hostname/node2", "ssh/ABC"}}

		assert.True(t, d.Add(a1, nil))
		assert.True(t, d.Add(a2, nil))
		assert.Len(t, d.Assets, 2)
	})

	t.Run("asset with no platform ids is rejected", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a1 := &inventory.Asset{PlatformIds: []string{}}

		assert.False(t, d.Add(a1, nil))
		assert.Len(t, d.Assets, 0)
	})

	t.Run("completely different ids are added", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a1 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}
		a2 := &inventory.Asset{PlatformIds: []string{"hostname/node3", "ssh/DEF"}}

		assert.True(t, d.Add(a1, nil))
		assert.True(t, d.Add(a2, nil))
		assert.Len(t, d.Assets, 2)
	})

	t.Run("subset of existing ids is rejected", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a1 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC", "cloud/xyz"}}
		a2 := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}

		assert.True(t, d.Add(a1, nil))
		assert.False(t, d.Add(a2, nil))
		assert.Len(t, d.Assets, 1)
	})

	t.Run("subset is evicted when superset arrives", func(t *testing.T) {
		d := &DiscoveredAssets{}
		fewer := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC"}}
		more := &inventory.Asset{PlatformIds: []string{"hostname/node1", "ssh/ABC", "cloud/xyz"}}

		// When the subset is added first, adding the superset evicts the subset.
		assert.True(t, d.Add(fewer, nil))
		assert.True(t, d.Add(more, nil))
		assert.Len(t, d.Assets, 1)
		assert.Equal(t, more.PlatformIds, d.Assets[0].Asset.PlatformIds)
	})

	t.Run("eviction is preserved when new asset is then rejected", func(t *testing.T) {
		d := &DiscoveredAssets{}
		small := &inventory.Asset{PlatformIds: []string{"id/1"}}
		big := &inventory.Asset{PlatformIds: []string{"id/1", "id/2", "id/3"}}
		mid := &inventory.Asset{PlatformIds: []string{"id/1", "id/2"}}

		assert.True(t, d.Add(small, nil))
		assert.True(t, d.Add(big, nil))
		// mid evicts small (subset), but is itself a subset of big → rejected.
		// small must stay evicted.
		assert.False(t, d.Add(mid, nil))
		assert.Len(t, d.Assets, 1)
		assert.Equal(t, big.PlatformIds, d.Assets[0].Asset.PlatformIds)
	})

	t.Run("no cross-asset conflation", func(t *testing.T) {
		d := &DiscoveredAssets{}
		a := &inventory.Asset{PlatformIds: []string{"hostname/X", "ssh/KEY1"}}
		b := &inventory.Asset{PlatformIds: []string{"hostname/Y", "ssh/KEY2"}}
		c := &inventory.Asset{PlatformIds: []string{"hostname/X", "ssh/KEY2"}}

		assert.True(t, d.Add(a, nil))
		assert.True(t, d.Add(b, nil))
		// C shares one ID with A and one with B, but is not a subset of either.
		assert.True(t, d.Add(c, nil))
		assert.Len(t, d.Assets, 3)
	})
}
