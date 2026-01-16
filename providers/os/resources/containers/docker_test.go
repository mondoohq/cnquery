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
	assert.NotEmpty(t, containers)

	// Check first container
	if len(containers) > 0 {
		c := containers[0]
		assert.NotEmpty(t, c.ID)
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.Image)
		assert.Equal(t, "docker", c.Runtime)
	}
}
