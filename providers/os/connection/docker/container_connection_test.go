// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/uuid"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	containerimage "go.mondoo.com/mql/v13/providers/os/connection/container/image"
	"go.mondoo.com/mql/v13/providers/os/connection/tar"
	"go.mondoo.com/mql/v13/providers/os/id/containerid"
)

// resolveRemoteImage returns the digest the registry currently advertises for
// imageRef. Tests use this so assertions stay valid when the upstream tag is
// rebuilt and the manifest digest changes.
func resolveRemoteImage(t *testing.T, imageRef string) string {
	t.Helper()
	ref, err := name.ParseReference(imageRef)
	require.NoError(t, err)
	desc, err := containerimage.GetImageDescriptor(ref)
	require.NoError(t, err)
	return desc.Digest.String()
}

// This test has an external dependency on the mirror.gcr.io registry.
// mirror.gcr.io is Google's anonymous pull-through cache for Docker Hub, so it
// avoids the docker-credential-gcloud helper that gcr.io triggers on CI runners.
// To test this specific case, we cannot use a stored image, we need to call remote.Get
func TestAssetNameForRemoteImages(t *testing.T) {
	const imageRef = "mirror.gcr.io/library/busybox:1.36.1"
	const imageRepo = "mirror.gcr.io/library/busybox"
	imgDigest := resolveRemoteImage(t, imageRef)

	var err error
	var conn *tar.Connection
	var asset *inventory.Asset
	retries := 3
	counter := 0

	config := &inventory.Config{
		Type: "docker-image",
		Host: imageRef,
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
	assert.Equal(t, imageRepo+"@"+containerid.ShortContainerImageID(imgDigest), asset.Name)
	assert.Contains(t, asset.PlatformIds, containerid.MondooContainerImageID(imgDigest))
}

// This test has an external dependency on the mirror.gcr.io registry.
// mirror.gcr.io is Google's anonymous pull-through cache for Docker Hub, so it
// avoids the docker-credential-gcloud helper that gcr.io triggers on CI runners.
// To test this specific case, we cannot use a stored image, we need to call remote.Get
func TestAssetNameForRemoteImages_DisableDelayedDiscovery(t *testing.T) {
	const imageRef = "mirror.gcr.io/library/busybox:1.36.1"
	const imageRepo = "mirror.gcr.io/library/busybox"
	imgDigest := resolveRemoteImage(t, imageRef)

	var err error
	var conn *tar.Connection
	var asset *inventory.Asset
	retries := 3
	counter := 0

	config := &inventory.Config{
		Type: "docker-image",
		Host: imageRef,
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
	assert.Equal(t, imageRepo+"@"+containerid.ShortContainerImageID(imgDigest), asset.Name)
	assert.Contains(t, asset.PlatformIds, containerid.MondooContainerImageID(imgDigest))
}

func fetchAndCreateImage(t *testing.T, ctx context.Context, dClient *client.Client, img string) container.CreateResponse {
	// If docker is not available, then skip the test.
	_, err := dClient.ServerVersion(ctx)
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
		err := dClient.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true})
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
