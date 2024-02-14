// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azcompute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
)

func TestCommandProviderLinux(t *testing.T) {
	conn, err := mock.New(0, "./testdata/metadata_linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := commandInstanceMetadata{conn, platform}
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/virtualMachines/examplevmname", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx", ident.AccountID)
}

func TestCommandProviderWindows(t *testing.T) {
	conn, err := mock.New(0, "./testdata/metadata_windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := commandInstanceMetadata{conn, platform}
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/virtualMachines/examplevmname", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx", ident.AccountID)
}
