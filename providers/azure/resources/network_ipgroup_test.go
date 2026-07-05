// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
	"github.com/stretchr/testify/assert"
)

func TestAzureNetworkSubResourceIDs(t *testing.T) {
	t.Run("nil slice yields nil", func(t *testing.T) {
		assert.Nil(t, azureNetworkSubResourceIDs(nil))
	})

	t.Run("skips nil entries and nil inner IDs", func(t *testing.T) {
		subs := []*network.SubResource{
			{ID: strPtr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/azureFirewalls/fw-a")},
			nil,
			{ID: nil},
			{ID: strPtr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/azureFirewalls/fw-b")},
		}
		got := azureNetworkSubResourceIDs(subs)
		assert.Equal(t, []string{
			"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/azureFirewalls/fw-a",
			"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/azureFirewalls/fw-b",
		}, got)
	})

	t.Run("all-nil slice yields nil", func(t *testing.T) {
		subs := []*network.SubResource{nil, {ID: nil}}
		assert.Nil(t, azureNetworkSubResourceIDs(subs))
	})
}
