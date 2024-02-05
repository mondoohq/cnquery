// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
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
	inventory := &inventory.Inventory{
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
	discoveredAssets, err := DiscoverAssets(context.Background(), inventory, nil, providers.NullRecording{})
	require.NoError(t, err)
	assert.Len(t, discoveredAssets.Assets, 3)
	assert.Len(t, discoveredAssets.Errors, 0)
	assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[0].Asset.ManagedBy)
	assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[1].Asset.ManagedBy)
	assert.Equal(t, "mondoo-operator-123", discoveredAssets.Assets[2].Asset.ManagedBy)
}
