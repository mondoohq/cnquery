// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gce_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
	"go.mondoo.com/cnquery/v10/providers/os/id/gce"
)

func TestCommandProviderLinux(t *testing.T) {
	conn, err := mock.New(0, "./testdata/metadata_linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := gce.NewCommandInstanceMetadata(conn, platform)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", ident.ProjectID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/6001244637815193808", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/instance-name", ident.PlatformMrn)
}

func TestCommandProviderWindows(t *testing.T) {
	conn, err := mock.New(0, "./testdata/metadata_windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := gce.NewCommandInstanceMetadata(conn, platform)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/5275377306317132843", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", ident.ProjectID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/instance-name", ident.PlatformMrn)
}
