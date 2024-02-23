// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
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

// TestDockerContainerConnection creates a new running container and tests the connection
func TestDockerContainerConnection(t *testing.T) {
	image := "docker.io/nginx:stable"
	ctx := context.Background()
	dClient, err := GetDockerClient()
	assert.NoError(t, err)

	// If docker is not available, then skip the test.
	_, err = dClient.ServerVersion(ctx)
	if err != nil {
		t.SkipNow()
	}

	responseBody, err := dClient.ImagePull(ctx, image, types.ImagePullOptions{})
	defer responseBody.Close()
	require.NoError(t, err)

	_, err = io.Copy(os.Stdout, responseBody)
	require.NoError(t, err)

	// Make sure the docker image is cleaned up
	defer func() {
		_, err := dClient.ImageRemove(ctx, image, types.ImageRemoveOptions{})
		require.NoError(t, err, "failed to cleanup pre-pulled docker image")
	}()

	cfg := &container.Config{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		StdinOnce:    false,
		Image:        image,
	}

	uuid := uuid.New()
	created, err := dClient.ContainerCreate(ctx, cfg, &container.HostConfig{}, &network.NetworkingConfig{}, &specs.Platform{}, uuid.String())
	require.NoError(t, err)

	require.NoError(t, dClient.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}))

	// Make sure the container is cleaned up
	defer func() {
		err := dClient.ContainerRemove(ctx, created.ID, types.ContainerRemoveOptions{Force: true})
		require.NoError(t, err)
	}()

	fmt.Println("inject: " + created.ID)
	conn, err := NewDockerContainerConnection(0, nil, nil)
	assert.NoError(t, err)

	cmd, err := conn.RunCommand("ls /")
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, 0, cmd.ExitStatus)
}
