// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/docker"
)

func TestDockerProcsList(t *testing.T) {
	img := "docker.io/nginx:stable"
	ctx := context.Background()
	dClient, err := docker.GetDockerClient()
	assert.NoError(t, err)

	// If docker is not available, then skip the test.
	_, err = dClient.ServerVersion(ctx)
	if err != nil {
		t.SkipNow()
	}

	responseBody, err := dClient.ImagePull(ctx, img, image.PullOptions{})
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
		_, err := dClient.ImageRemove(ctx, img, image.RemoveOptions{Force: true})
		// ignore error, worst case is that the image is not removed but parallel tests may fail otherwise
		fmt.Printf("failed to cleanup pre-pulled docker image: %v", err)
	}()

	cfg := &container.Config{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		StdinOnce:    false,
		Image:        img,
	}

	uuidVal := uuid.New()
	created, err := dClient.ContainerCreate(ctx, cfg, &container.HostConfig{}, &network.NetworkingConfig{}, &specs.Platform{}, uuidVal.String())
	require.NoError(t, err)

	require.NoError(t, dClient.ContainerStart(ctx, created.ID, container.StartOptions{}))

	// Make sure the container is cleaned up
	defer func() {
		err := dClient.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true})
		require.NoError(t, err)
	}()

	fmt.Println("inject: " + created.ID)
	conn, err := docker.NewContainerConnection(0, &inventory.Config{
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

	pMan, err := ResolveManager(conn)
	assert.NoError(t, err)

	proc, err := pMan.Process(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, proc)
}
