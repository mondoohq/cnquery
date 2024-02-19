// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
)

// This test has an external dependency on the gcr.io registry
// To test this specific case, we cannot use a stored image, we need to call remote.Get
func TestAssetNameForRemoteImages(t *testing.T) {
	var err error
	var conn *tar.TarConnection
	var asset *inventory.Asset
	retries := 3
	counter := 0

	for {
		config := &inventory.Config{
			Type: "docker-image",
			Host: "gcr.io/google-containers/busybox:1.27.2",
		}
		asset = &inventory.Asset{
			Connections: []*inventory.Config{config},
		}
		conn, err = NewDockerContainerImageConnection(0, config, asset)
		if counter > retries || (err == nil && conn != nil) {
			break
		}
		counter++
	}
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, "gcr.io/google-containers/busybox@545e6a6310a2", asset.Name)
	assert.Contains(t, asset.PlatformIds, "//platformid.api.mondoo.app/runtime/docker/images/545e6a6310a27636260920bc07b994a299b6708a1b26910cfefd335fdfb60d2b")
}
