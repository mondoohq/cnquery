// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package aws_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
)

func TestInventory(t *testing.T) {
	inventoryContent := `
apiVersion: v1
kind: Inventory
metadata:
  name: aws-inventory
spec:
  assets:
    - id: account-1
      connections:
        - backend: aws
          options:
            profile: mondoo-dev
          discover:
            targets:
              - "accounts"
      annotations:
        owner: user@example.com
    - id: account-2
      connections:
        - backend: aws
          options:
            profile: mondoo-demo
          discover:
            targets:
              - "accounts"
      annotations:
        owner: user@example.com% 
`
	inv, err := v1.InventoryFromYAML([]byte(inventoryContent))
	require.NoError(t, err)

	im, err := inventory.New(inventory.WithInventory(inv))

	ctx := context.Background()
	assetErrors := im.Resolve(ctx)
	assert.True(t, len(assetErrors) == 0)

	assetList := im.GetAssets()
	assert.True(t, len(assetList) == 2)
}
