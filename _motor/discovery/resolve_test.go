// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package discovery_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/discovery"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
)

func TestResolverWithAssetName(t *testing.T) {
	inventory := &v1.Inventory{
		Spec: &v1.InventorySpec{
			Assets: []*v1.Asset{
				{
					Name: "test",
					Connections: []*v1.Config{
						{
							Type: "local",
						},
					},
				},
				{
					Connections: []*v1.Config{
						{
							Type: "mock",
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
