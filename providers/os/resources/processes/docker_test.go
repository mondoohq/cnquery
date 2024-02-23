// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes

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
	"go.mondoo.com/cnquery/v10/providers/os/connection/docker"
)

func TestDockerProcsList(t *testing.T) {
	image := "docker.io/nginx:stable"
	ctx := context.Background()
	dClient, err := docker.GetDockerClient()
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
	conn, err := docker.NewDockerContainerConnection(0, &inventory.Config{
		Host: created.ID,
	}, &inventory.Asset{
		// for the test we need to set the platform
		Platform: &inventory.Platform{
			Name:    "debian",
			Version: "11",
			Family:  []string{"debian", "linux"},
		},
	})
	assert.NoError(t, err)

	pMan, err := ResolveManager(conn)
	assert.NoError(t, err)

	proc, err := pMan.Process(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, proc)
}
