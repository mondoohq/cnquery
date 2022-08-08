package discovery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery"
	v1 "go.mondoo.io/mondoo/motor/inventory/v1"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestResolverWithAssetName(t *testing.T) {
	inventory := &v1.Inventory{
		Spec: &v1.InventorySpec{
			Assets: []*asset.Asset{
				{
					Name: "test",
					Connections: []*providers.TransportConfig{
						{
							Backend: providers.TransportBackend_CONNECTION_LOCAL_OS,
						},
					},
				},
				{
					Connections: []*providers.TransportConfig{
						{
							Backend: providers.TransportBackend_CONNECTION_MOCK,
							Options: map[string]string{
								"path": "./testdata/mock.toml",
							},
						},
					},
				},
			},
		},
	}

	resolved := discovery.ResolveAssets(inventory.Spec.Assets, nil, nil)
	assert.Equal(t, 2, len(resolved.Assets))
	assert.Equal(t, "test", resolved.Assets[0].Name)
	assert.Equal(t, "testmachine", resolved.Assets[1].Name)
	assert.Equal(t, 0, len(resolved.Errors))
}
