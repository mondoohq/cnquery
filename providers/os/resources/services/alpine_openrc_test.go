// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestManagerAlpineImage(t *testing.T) {
	mock, err := mock.New(0, "./testdata/alpine-image.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "alpine",
		},
	})
	require.NoError(t, err)

	mm, err := ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(serviceList))

	assert.Contains(t, serviceList, &Service{
		Name:      "agetty",
		Running:   false, // service will not run, since its a container image
		Enabled:   true,
		Installed: true,
		Type:      "openrc",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
	})
}

func TestManagerAlpineContainer(t *testing.T) {
	mock, err := mock.New(0, "./testdata/alpine-container.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "alpine",
		},
	})
	require.NoError(t, err)

	mm, err := ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(serviceList))

	assert.Contains(t, serviceList, &Service{
		Name:      "agetty",
		Running:   true, // here this service is actually running
		Enabled:   true,
		Installed: true,
		Type:      "openrc",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
	})
}
