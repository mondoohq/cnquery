package processes

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/container/docker_engine"
)

func TestDockerProcsList(t *testing.T) {
	image := "docker.io/nginx:stable-alpine"
	ctx := context.Background()
	dClient, err := docker_engine.GetDockerClient()
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
	created, err := dClient.ContainerCreate(ctx, cfg, &container.HostConfig{}, &network.NetworkingConfig{}, uuid.String())
	require.NoError(t, err)

	require.NoError(t, dClient.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}))

	// Make sure the container is cleaned up
	defer func() {
		err := dClient.ContainerRemove(ctx, created.ID, types.ContainerRemoveOptions{Force: true})
		require.NoError(t, err)
	}()

	transport, err := docker_engine.New(created.ID)
	assert.NoError(t, err)

	motor, err := motor.New(transport)
	assert.NoError(t, err)

	pMan, err := ResolveManager(motor)
	assert.NoError(t, err)

	procs, err := pMan.List()
	assert.NoError(t, err)
	assert.NotEmpty(t, procs)
}
