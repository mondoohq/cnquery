// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
	"github.com/stretchr/testify/assert"
)

func TestAddressPrefixItemsToAny(t *testing.T) {
	t.Run("nil slice yields empty", func(t *testing.T) {
		assert.Equal(t, []any{}, addressPrefixItemsToAny(nil))
	})

	t.Run("skips nil items and nil prefixes", func(t *testing.T) {
		items := []*network.AddressPrefixItem{
			{AddressPrefix: strPtr("10.0.0.0/8")},
			nil,
			{AddressPrefix: nil},
			{AddressPrefix: strPtr("Internet")},
		}
		assert.Equal(t, []any{"10.0.0.0/8", "Internet"}, addressPrefixItemsToAny(items))
	})
}

func TestStrPtrsToAny(t *testing.T) {
	t.Run("nil slice yields empty", func(t *testing.T) {
		assert.Equal(t, []any{}, strPtrsToAny(nil))
	})

	t.Run("skips nil elements", func(t *testing.T) {
		assert.Equal(t, []any{"80", "443"}, strPtrsToAny([]*string{strPtr("80"), nil, strPtr("443")}))
	})
}
