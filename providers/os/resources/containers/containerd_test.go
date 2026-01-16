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

func TestContainerdManager_List(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "unix"},
		},
	}, mock.WithPath("./testdata/containerd.toml"))
	require.NoError(t, err)

	cm := &ContainerdManager{conn: mock}

	containers, err := cm.List()
	require.NoError(t, err)

	// Should return 2 containers from the mocked output
	assert.Len(t, containers, 2)

	// Test first container
	c1 := containers[0]
	assert.Equal(t, "test-container", c1.ID)
	assert.Equal(t, "test-container", c1.Name)
	assert.Equal(t, "docker.io/library/nginx:latest", c1.Image)
	assert.Equal(t, "unknown", c1.State)
	assert.Equal(t, "unknown", c1.Status)
	assert.Equal(t, "containerd", c1.Runtime)

	// Test second container
	c2 := containers[1]
	assert.Equal(t, "database_service_v2", c2.ID)
	assert.Equal(t, "database_service_v2", c2.Name)
	assert.Equal(t, "docker.io/library/postgres:latest", c2.Image)
	assert.Equal(t, "unknown", c2.State)
	assert.Equal(t, "unknown", c2.Status)
	assert.Equal(t, "containerd", c2.Runtime)
}

func TestContainerdManager_IsAvailable(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "unix"},
		},
	}, mock.WithPath("./testdata/containerd.toml"))
	require.NoError(t, err)

	cm := &ContainerdManager{conn: mock}

	// Should detect containerd is available via ctr version command
	assert.True(t, cm.IsAvailable())
}
