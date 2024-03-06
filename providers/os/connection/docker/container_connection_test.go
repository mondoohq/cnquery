// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
)

// This test has an external dependency on the gcr.io registry
// To test this specific case, we cannot use a stored image, we need to call remote.Get
func TestAssetNameForRemoteImages(t *testing.T) {
	var err error
	var conn *tar.Connection
	var asset *inventory.Asset
	retries := 3
	counter := 0

	config := &inventory.Config{
		Type: "docker-image",
		Host: "gcr.io/google-containers/busybox:1.27.2",
	}
	asset = &inventory.Asset{
		Connections: []*inventory.Config{config},
	}

	for {
		conn, err = NewContainerImageConnection(0, config, asset)
		if counter > retries || (err == nil && conn != nil) {
			break
		}
		counter++
	}
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.True(t, config.DelayDiscovery)
	assert.Equal(t, "gcr.io/google-containers/busybox@545e6a6310a2", asset.Name)
	assert.Contains(t, asset.PlatformIds, "//platformid.api.mondoo.app/runtime/docker/images/545e6a6310a27636260920bc07b994a299b6708a1b26910cfefd335fdfb60d2b")
}

// This test has an external dependency on the gcr.io registry
// To test this specific case, we cannot use a stored image, we need to call remote.Get
func TestAssetNameForRemoteImages_DisableDelayedDiscovery(t *testing.T) {
	var err error
	var conn *tar.Connection
	var asset *inventory.Asset
	retries := 3
	counter := 0

	config := &inventory.Config{
		Type: "docker-image",
		Host: "gcr.io/google-containers/busybox:1.27.2",
		Options: map[string]string{
			plugin.DISABLE_DELAYED_DISCOVERY_OPTION: "true",
		},
	}
	asset = &inventory.Asset{
		Connections: []*inventory.Config{config},
	}

	for {
		conn, err = NewContainerImageConnection(0, config, asset)
		if counter > retries || (err == nil && conn != nil) {
			break
		}
		counter++
	}
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.False(t, config.DelayDiscovery)
	assert.Equal(t, "gcr.io/google-containers/busybox@545e6a6310a2", asset.Name)
	assert.Contains(t, asset.PlatformIds, "//platformid.api.mondoo.app/runtime/docker/images/545e6a6310a27636260920bc07b994a299b6708a1b26910cfefd335fdfb60d2b")
}

func fetchAndCreateImage(t *testing.T, ctx context.Context, dClient *client.Client, image string) container.CreateResponse {
	// If docker is not available, then skip the test.
	_, err := dClient.ServerVersion(ctx)
	if err != nil {
		t.SkipNow()
	}

	responseBody, err := dClient.ImagePull(ctx, image, types.ImagePullOptions{})
	defer func() {
		err = responseBody.Close()
		if err != nil {
			panic(err)
		}
	}()
	require.NoError(t, err)

	_, err = io.Copy(os.Stdout, responseBody)
	require.NoError(t, err)

	// Make sure the docker image is cleaned up
	defer func() {
		_, err := dClient.ImageRemove(ctx, image, types.ImageRemoveOptions{
			Force: true,
		})
		// ignore error, worst case is that the image is not removed but parallel tests may fail otherwise
		fmt.Printf("failed to cleanup pre-pulled docker image: %v", err)
	}()

	cfg := &container.Config{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		StdinOnce:    false,
		Image:        image,
	}

	uuidVal := uuid.New()
	created, err := dClient.ContainerCreate(ctx, cfg, &container.HostConfig{}, &network.NetworkingConfig{}, &specs.Platform{}, uuidVal.String())
	require.NoError(t, err)

	require.NoError(t, dClient.ContainerStart(ctx, created.ID, container.StartOptions{}))

	return created
}

// TestDockerContainerConnection creates a new running container and tests the connection
func TestDockerContainerConnection(t *testing.T) {
	ctx := context.Background()
	image := "docker.io/nginx:stable"
	dClient, err := GetDockerClient()
	assert.NoError(t, err)
	created := fetchAndCreateImage(t, ctx, dClient, image)

	// Make sure the container is cleaned up
	defer func() {
		err := dClient.ContainerRemove(ctx, created.ID, container.RemoveOptions{
			Force: true,
		})
		require.NoError(t, err)
	}()

	fmt.Println("inject: " + created.ID)
	conn, err := NewContainerConnection(0, &inventory.Config{
		Host: created.ID,
	}, &inventory.Asset{
		// for the test we need to set the platform
		Platform: &inventory.Platform{
			Name:    "debian",
			Version: "11",
			Family:  []string{"debian", "linux"},
		},
	})
	require.NoError(t, err)

	cmd, err := conn.RunCommand("ls /")
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, 0, cmd.ExitStatus)
}
