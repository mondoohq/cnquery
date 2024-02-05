// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers"
	inventory "go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/pack1",
		"namespace2/pack2",
		"//registry.mondoo.com/namespace/namespace3/querypacks/pack3",
	}

	// when
	preprocessed := preprocessQueryPackFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/querypacks/pack1",
		"//registry.mondoo.com/namespace/namespace2/querypacks/pack2",
		"//registry.mondoo.com/namespace/namespace3/querypacks/pack3",
	}, preprocessed)
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
		inv.Spec.Assets[0].Category = inventory.AssetCategory_CATEGORY_CICD
		require.NoError(t, os.Setenv("KUBERNETES_ADMISSION_CONTROLLER", "true"))
		discoveredAssets, err := DiscoverAssets(context.Background(), inv, nil, providers.NullRecording{})
		require.NoError(t, err)

		for _, asset := range discoveredAssets.Assets {
			require.Contains(t, asset.Asset.Labels, "mondoo.com/exec-environment")
			assert.Equal(t, "k8s.mondoo.com", asset.Asset.Labels["mondoo.com/exec-environment"])
		}
	})
}
