package core_test

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
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/inventory"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/container/docker_engine"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/testutils"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestResource_Platform(t *testing.T) {
	t.Run("platform info", func(t *testing.T) {
		res := x.TestQuery(t, "platform")
		assert.NotEmpty(t, res)
	})

	t.Run("platform name", func(t *testing.T) {
		res := x.TestQuery(t, "platform.name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "arch", res[0].Data.Value)
	})
}

// TODO: can we move this test somewhere else, a bit closer to the shell-level tests?
func TestDockerContainer_Platform(t *testing.T) {
	// to avoid this breaking if the nginx:stable ever changes its platform
	// pin it to a concrete digest that resolves to a debian 11.5
	image := "docker.io/nginx@sha256:c79f4fe604e3fe77cb5142e9747da3132d252af21fbb9a9d294fa2128499a8f1"
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
	created, err := dClient.ContainerCreate(ctx, cfg, &container.HostConfig{}, &network.NetworkingConfig{}, &specs.Platform{}, uuid.String())
	require.NoError(t, err)
	require.NoError(t, dClient.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}))

	// Make sure the container is cleaned up
	defer func() {
		err := dClient.ContainerRemove(ctx, created.ID, types.ContainerRemoveOptions{Force: true})
		require.NoError(t, err)
	}()
	connAsset := &asset.Asset{
		Options:     map[string]string{},
		Connections: []*providers.Config{},
	}

	connection := &providers.Config{
		Backend: providers.ProviderType_DOCKER_ENGINE_CONTAINER,
		Host:    created.ID,
	}
	connAsset.Connections = append(connAsset.Connections, connection)
	im, err := inventory.New(inventory.WithAssets([]*asset.Asset{connAsset}))
	require.NoError(t, err)

	assetErrors := im.Resolve(ctx)
	require.Empty(t, assetErrors)
	assetList := im.GetAssets()
	container := assetList[0]
	motor, err := resolver.OpenAssetConnection(ctx, container, nil, false)
	require.NoError(t, err)

	tester := testutils.InitTester(motor, core.Registry)
	name := tester.TestQuery(t, "platform.name")
	release := tester.TestQuery(t, "platform.release")
	assert.Equal(t, "debian", name[0].Data.Value)
	assert.Equal(t, "11.5", release[0].Data.Value)
}
