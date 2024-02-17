// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers"
	inventory "go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func TestDiscoveredAssets_Add(t *testing.T) {
	d := &DiscoveredAssets{
		platformIds: map[string]struct{}{},
		Assets:      []*AssetWithRuntime{},
		Errors:      []*AssetWithError{},
	}
	asset := &inventory.Asset{
		PlatformIds: []string{"platform1"},
	}
	runtime := &providers.Runtime{}

	assert.True(t, d.Add(asset, runtime))
	assert.Len(t, d.Assets, 1)
	assert.Len(t, d.Errors, 0)

	// Make sure adding duplicates is not possible
	assert.False(t, d.Add(asset, runtime))
	assert.Len(t, d.Assets, 1)
	assert.Len(t, d.Errors, 0)
}

func TestDiscoveredAssets_Add_MultiplePlatformIDs(t *testing.T) {
	d := &DiscoveredAssets{
		platformIds: map[string]struct{}{},
		Assets:      []*AssetWithRuntime{},
		Errors:      []*AssetWithError{},
	}
	asset := &inventory.Asset{
		PlatformIds: []string{"platform1", "platform2"},
	}
	runtime := &providers.Runtime{}

	assert.True(t, d.Add(asset, runtime))
	assert.Len(t, d.Assets, 1)
	assert.Len(t, d.Errors, 0)

	// Make sure adding duplicates is not possible
	assert.False(t, d.Add(&inventory.Asset{
		PlatformIds: []string{"platform3", asset.PlatformIds[0]},
	}, runtime))
	assert.Len(t, d.Assets, 1)
	assert.Len(t, d.Errors, 0)
}

func TestDiscoveredAssets_GetAssetsByPlatformID(t *testing.T) {
	d := &DiscoveredAssets{
		platformIds: map[string]struct{}{},
		Assets:      []*AssetWithRuntime{},
		Errors:      []*AssetWithError{},
	}

	allPlatformIds := []string{}
	for i := 0; i < 10; i++ {
		pId := fmt.Sprintf("platform1%d", i)
		allPlatformIds = append(allPlatformIds, pId)
		asset := &inventory.Asset{
			PlatformIds: []string{pId},
		}
		runtime := &providers.Runtime{}

		assert.True(t, d.Add(asset, runtime))
	}
	assert.Len(t, d.Assets, 10)

	// Make sure adding duplicates is not possible
	assets := d.GetAssetsByPlatformID(allPlatformIds[0])
	assert.Len(t, assets, 1)
	assert.Equal(t, allPlatformIds[0], assets[0].Asset.PlatformIds[0])
}

func TestDiscoveredAssets_GetAssetsByPlatformID_Empty(t *testing.T) {
	d := &DiscoveredAssets{
		platformIds: map[string]struct{}{},
		Assets:      []*AssetWithRuntime{},
		Errors:      []*AssetWithError{},
	}

	allPlatformIds := []string{}
	for i := 0; i < 10; i++ {
		pId := fmt.Sprintf("platform1%d", i)
		allPlatformIds = append(allPlatformIds, pId)
		asset := &inventory.Asset{
			PlatformIds: []string{pId},
		}
		runtime := &providers.Runtime{}

		assert.True(t, d.Add(asset, runtime))
	}
	assert.Len(t, d.Assets, 10)

	// Make sure adding duplicates is not possible
	assets := d.GetAssetsByPlatformID("")
	assert.Len(t, assets, 10)
	platformIds := []string{}
	for _, a := range assets {
		platformIds = append(platformIds, a.Asset.PlatformIds[0])
	}
	assert.ElementsMatch(t, allPlatformIds, platformIds)
}

func TestDiscoverAssets(t *testing.T) {
	getInventory := func() *inventory.Inventory {
		return &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/2pods.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"auto"},
								},
							},
						},
						ManagedBy: "mondoo-operator-123",
					},
				},
			},
		}
	}

	t.Run("normal", func(t *testing.T) {
		inv := getInventory()
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)
		assert.Len(t, discoveredAssets.Assets, 3)
		assert.Len(t, discoveredAssets.Errors, 0)
		assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[0].Asset.ManagedBy)
		assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[1].Asset.ManagedBy)
		assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[2].Asset.ManagedBy)
	})

	t.Run("with duplicate root assets", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets = append(inv.Spec.Assets, inv.Spec.Assets[0])
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		// Make sure no duplicates are returned
		assert.Len(t, discoveredAssets.Assets, 3)
		assert.Len(t, discoveredAssets.Errors, 0)
	})

	t.Run("with duplicate discovered assets", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets[0].Connections[0].Options["path"] = "./testdata/3pods_with_duplicate.yaml"
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		// Make sure no duplicates are returned
		assert.Len(t, discoveredAssets.Assets, 3)
		assert.Len(t, discoveredAssets.Errors, 0)
	})

	t.Run("copy root asset annotations", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets[0].Annotations = map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		for _, asset := range discoveredAssets.Assets {
			for k, v := range inv.Spec.Assets[0].Annotations {
				require.Contains(t, asset.Asset.Annotations, k)
				assert.Equal(t, v, asset.Asset.Annotations[k])
			}
		}
	})

	t.Run("copy root asset managedBy", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets[0].ManagedBy = "managed-by-test"
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		for _, asset := range discoveredAssets.Assets {
			assert.Equal(t, inv.Spec.Assets[0].ManagedBy, asset.Asset.ManagedBy)
		}
	})

	t.Run("set ci/cd labels", func(t *testing.T) {
		inv := getInventory()

		val, isSet := os.LookupEnv("GITHUB_ACTION")
		defer func() {
			if isSet {
				require.NoError(t, os.Setenv("GITHUB_ACTION", val))
			} else {
				require.NoError(t, os.Unsetenv("GITHUB_ACTION"))
			}
		}()
		inv.Spec.Assets[0].Category = inventory.AssetCategory_CATEGORY_CICD
		require.NoError(t, os.Setenv("GITHUB_ACTION", "go-test"))
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		for _, asset := range discoveredAssets.Assets {
			require.Contains(t, asset.Asset.Labels, "mondoo.com/exec-environment")
			assert.Equal(t, "actions.github.com", asset.Asset.Labels["mondoo.com/exec-environment"])
		}
	})

	t.Run("set ci/cd labels for scannable root assets", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets[0].Connections[0].Type = "local"

		val, isSet := os.LookupEnv("GITHUB_ACTION")
		defer func() {
			if isSet {
				require.NoError(t, os.Setenv("GITHUB_ACTION", val))
			} else {
				require.NoError(t, os.Unsetenv("GITHUB_ACTION"))
			}
		}()
		inv.Spec.Assets[0].Category = inventory.AssetCategory_CATEGORY_CICD
		require.NoError(t, os.Setenv("GITHUB_ACTION", "go-test"))
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		for _, asset := range discoveredAssets.Assets {
			require.Contains(t, asset.Asset.Labels, "mondoo.com/exec-environment")
			assert.Equal(t, "actions.github.com", asset.Asset.Labels["mondoo.com/exec-environment"])
		}
	})

	t.Run("scannable root asset", func(t *testing.T) {
		inv := getInventory()
		inv.Spec.Assets[0].Connections[0].Type = "local"

		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)
		assert.Len(t, discoveredAssets.Assets, 1)
	})
}
