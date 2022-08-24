package discovery_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestResolverWithAssetName(t *testing.T) {
	inventory := &v1.Inventory{
		Spec: &v1.InventorySpec{
			Assets: []*asset.Asset{
				{
					Name: "test",
					Connections: []*providers.Config{
						{
							Backend: providers.ProviderType_LOCAL_OS,
						},
					},
				},
				{
					Connections: []*providers.Config{
						{
							Backend: providers.ProviderType_MOCK,
							Options: map[string]string{
								"path": "./testdata/mock.toml",
							},
						},
					},
				},
			},
		},
	}

	resolved := discovery.ResolveAssets(context.Background(), inventory.Spec.Assets, nil, nil)
	assert.Equal(t, 2, len(resolved.Assets))
	assert.Equal(t, "test", resolved.Assets[0].Name)
	assert.Equal(t, "testmachine", resolved.Assets[1].Name)
	assert.Equal(t, 0, len(resolved.Errors))
}
