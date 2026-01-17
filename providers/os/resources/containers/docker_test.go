// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package containers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
)

func TestDockerManager_List(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "unix"},
		},
	}, mock.WithPath("./testdata/docker.toml"))
	require.NoError(t, err)

	dm := &DockerManager{conn: mock}

	containers, err := dm.List()
	require.NoError(t, err)

	// Should return all 3 containers from the mocked output
	assert.Len(t, containers, 3)

	// Test first container (exited state)
	c1 := containers[0]
	assert.Equal(t, "fc6f8b5ccaba", c1.ID)
	assert.Equal(t, "reverent_lewin", c1.Name)
	assert.Equal(t, "amazonlinux:2023", c1.Image)
	assert.Equal(t, "exited", c1.State)
	assert.Equal(t, "Exited (0) 10 hours ago", c1.Status)
	assert.Equal(t, "docker", c1.Runtime)
	assert.Contains(t, c1.Labels, "desktop.docker.io/ports.scheme")
	assert.Equal(t, "v2", c1.Labels["desktop.docker.io/ports.scheme"])

	// Test second container (running state)
	c2 := containers[1]
	assert.Equal(t, "2fe1c726e5bb", c2.ID)
	assert.Equal(t, "hopeful_keldysh", c2.Name)
	assert.Equal(t, "amazonlinux:2", c2.Image)
	assert.Equal(t, "running", c2.State)
	assert.Equal(t, "Up 10 hours", c2.Status)
	assert.Equal(t, "docker", c2.Runtime)

	// Test third container (created state)
	c3 := containers[2]
	assert.Equal(t, "2a13e2b1a2ee", c3.ID)
	assert.Equal(t, "jolly_villani", c3.Name)
	assert.Equal(t, "ghcr.io/gardenlinux/gardenlinux:latest", c3.Image)
	assert.Equal(t, "created", c3.State)
	assert.Equal(t, "Created", c3.Status)
	assert.Equal(t, "docker", c3.Runtime)
}

func TestDockerManager_ListRunningOnly(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "unix"},
		},
	}, mock.WithPath("./testdata/docker.toml"))
	require.NoError(t, err)

	dm := &DockerManager{conn: mock}

	containers, err := dm.List()
	require.NoError(t, err)

	// Filter for running containers only
	var runningContainers []*OSContainer
	for _, c := range containers {
		if c.State == "running" {
			runningContainers = append(runningContainers, c)
		}
	}

	// Should only have 1 running container
	assert.Len(t, runningContainers, 1)
	assert.Equal(t, "2fe1c726e5bb", runningContainers[0].ID)
	assert.Equal(t, "hopeful_keldysh", runningContainers[0].Name)
	assert.Equal(t, "running", runningContainers[0].State)
}
