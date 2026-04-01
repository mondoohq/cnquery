// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func newTestExplorer(assets ...*TrackedAsset) *AssetExplorer {
	return &AssetExplorer{
		allAssets: assets,
	}
}

func TestAssetExplorerDiscovered(t *testing.T) {
	discovered := &TrackedAsset{
		Asset: &inventory.Asset{Name: "child1"},
		State: AssetDiscovered,
	}
	connected := &TrackedAsset{
		Asset: &inventory.Asset{Name: "root"},
		State: AssetConnected,
	}
	closed := &TrackedAsset{
		Asset: &inventory.Asset{Name: "old"},
		State: AssetClosed,
	}

	e := newTestExplorer(discovered, connected, closed)
	result := e.Discovered()
	assert.Len(t, result, 1)
	assert.Equal(t, "child1", result[0].Asset.Name)
}

func TestAssetExplorerConnected(t *testing.T) {
	discovered := &TrackedAsset{
		Asset: &inventory.Asset{Name: "child1"},
		State: AssetDiscovered,
	}
	connected := &TrackedAsset{
		Asset: &inventory.Asset{Name: "root"},
		State: AssetConnected,
	}

	e := newTestExplorer(discovered, connected)
	result := e.Connected()
	assert.Len(t, result, 1)
	assert.Equal(t, "root", result[0].Asset.Name)
}

func TestAssetExplorerCloseAsset(t *testing.T) {
	t.Run("close connected asset", func(t *testing.T) {
		asset := &TrackedAsset{
			Asset: &inventory.Asset{Name: "root"},
			State: AssetConnected,
			// Runtime is nil in tests since we don't have a real provider
		}
		e := newTestExplorer(asset)

		err := e.CloseAsset(asset)
		require.NoError(t, err)
		assert.Equal(t, AssetClosed, asset.State)
		assert.Nil(t, asset.Runtime)
	})

	t.Run("close discovered asset fails", func(t *testing.T) {
		asset := &TrackedAsset{
			Asset: &inventory.Asset{Name: "child"},
			State: AssetDiscovered,
		}
		e := newTestExplorer(asset)

		err := e.CloseAsset(asset)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("close unknown asset fails", func(t *testing.T) {
		e := newTestExplorer()
		unknown := &TrackedAsset{
			Asset: &inventory.Asset{Name: "unknown"},
			State: AssetConnected,
		}

		err := e.CloseAsset(unknown)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not tracked")
	})
}

func TestAssetExplorerShutdown(t *testing.T) {
	connected1 := &TrackedAsset{
		Asset: &inventory.Asset{Name: "a"},
		State: AssetConnected,
	}
	connected2 := &TrackedAsset{
		Asset: &inventory.Asset{Name: "b"},
		State: AssetConnected,
	}
	discovered := &TrackedAsset{
		Asset: &inventory.Asset{Name: "c"},
		State: AssetDiscovered,
	}

	e := newTestExplorer(connected1, connected2, discovered)
	e.Shutdown()

	assert.Equal(t, AssetClosed, connected1.State)
	assert.Equal(t, AssetClosed, connected2.State)
	assert.Equal(t, AssetDiscovered, discovered.State) // unchanged
}

func TestAssetExplorerConnectAlreadyConnected(t *testing.T) {
	child1 := &TrackedAsset{
		Asset: &inventory.Asset{Name: "child1"},
		State: AssetDiscovered,
	}
	parent := &TrackedAsset{
		Asset:    &inventory.Asset{Name: "root", PlatformIds: []string{"id/1"}},
		State:    AssetConnected,
		Children: []*TrackedAsset{child1},
	}

	e := newTestExplorer(parent, child1)
	connected, err := e.Connect(parent)
	require.NoError(t, err)
	assert.Equal(t, parent, connected)
	assert.Len(t, connected.Children, 1)
	assert.Equal(t, "child1", connected.Children[0].Asset.Name)
}

func TestAssetExplorerConnectClosedAsset(t *testing.T) {
	asset := &TrackedAsset{
		Asset: &inventory.Asset{Name: "closed"},
		State: AssetClosed,
	}
	e := newTestExplorer(asset)

	_, err := e.Connect(asset)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestAssetExplorerConnectUnknownAsset(t *testing.T) {
	e := newTestExplorer()
	unknown := &TrackedAsset{
		Asset: &inventory.Asset{Name: "unknown"},
		State: AssetDiscovered,
	}

	_, err := e.Connect(unknown)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked")
}

func TestAssetExplorerDedup(t *testing.T) {
	t.Run("new asset that is subset of existing gets closed", func(t *testing.T) {
		existing := &TrackedAsset{
			Asset: &inventory.Asset{Name: "big", PlatformIds: []string{"id/1", "id/2", "id/3"}},
			State: AssetConnected,
		}
		newAsset := &TrackedAsset{
			Asset: &inventory.Asset{Name: "small", PlatformIds: []string{"id/1", "id/2"}},
			State: AssetConnected,
		}

		e := newTestExplorer(existing, newAsset)
		e.dedup(newAsset)

		assert.Equal(t, AssetClosed, newAsset.State)
		assert.Equal(t, AssetConnected, existing.State)
	})

	t.Run("existing asset that is subset of new gets evicted", func(t *testing.T) {
		existing := &TrackedAsset{
			Asset: &inventory.Asset{Name: "small", PlatformIds: []string{"id/1"}},
			State: AssetConnected,
		}
		newAsset := &TrackedAsset{
			Asset: &inventory.Asset{Name: "big", PlatformIds: []string{"id/1", "id/2"}},
			State: AssetConnected,
		}

		e := newTestExplorer(existing, newAsset)
		e.dedup(newAsset)

		assert.Equal(t, AssetClosed, existing.State)
		assert.Equal(t, AssetConnected, newAsset.State)
	})

	t.Run("unrelated assets coexist", func(t *testing.T) {
		a := &TrackedAsset{
			Asset: &inventory.Asset{Name: "a", PlatformIds: []string{"id/1"}},
			State: AssetConnected,
		}
		b := &TrackedAsset{
			Asset: &inventory.Asset{Name: "b", PlatformIds: []string{"id/2"}},
			State: AssetConnected,
		}

		e := newTestExplorer(a, b)
		e.dedup(b)

		assert.Equal(t, AssetConnected, a.State)
		assert.Equal(t, AssetConnected, b.State)
	})

	t.Run("assets with no platform IDs are skipped", func(t *testing.T) {
		a := &TrackedAsset{
			Asset: &inventory.Asset{Name: "gateway"},
			State: AssetConnected,
		}

		e := newTestExplorer(a)
		e.dedup(a) // should not panic or change state

		assert.Equal(t, AssetConnected, a.State)
	})
}

func TestAssetExplorerErrors(t *testing.T) {
	e := newTestExplorer()
	e.errors = []*AssetWithError{
		{Asset: &inventory.Asset{Name: "bad"}, Err: assert.AnError},
	}

	errs := e.Errors()
	assert.Len(t, errs, 1)
	assert.Equal(t, "bad", errs[0].Asset.Name)
}

func TestTrackedAssetParentChain(t *testing.T) {
	root := &TrackedAsset{
		Asset: &inventory.Asset{Name: "root", PlatformIds: []string{"root/1"}},
		State: AssetConnected,
	}
	mid := &TrackedAsset{
		Asset:  &inventory.Asset{Name: "mid"},
		State:  AssetConnected,
		Parent: root,
	}
	leaf := &TrackedAsset{
		Asset:  &inventory.Asset{Name: "leaf"},
		State:  AssetDiscovered,
		Parent: mid,
	}

	e := newTestExplorer(root, mid, leaf)
	foundRoot := e.findRootAsset(leaf)
	assert.Equal(t, "root", foundRoot.Name)
}
