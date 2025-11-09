// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
)

func TestManagerAlpineImage(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "alpine",
		},
	}, mock.WithPath("./testdata/alpine-image.toml"))
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
		Path:      "/etc/init.d/agetty",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
		Path:      "/etc/init.d/urandom",
	})
}

func TestManagerAlpineContainer(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "alpine",
		},
	}, mock.WithPath("./testdata/alpine-container.toml"))
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
		Path:      "/etc/init.d/agetty",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
		Path:      "/etc/init.d/urandom",
	})
}

func TestManagerGentoo(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "gentoo",
		},
	}, mock.WithPath("./testdata/gentoo.toml"))
	require.NoError(t, err)

	mm, err := ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(serviceList))

	assert.Contains(t, serviceList, &Service{
		Name:      "agetty",
		Running:   true,
		Enabled:   true,
		Installed: true,
		Type:      "openrc",
		Path:      "/etc/init.d/agetty",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "sysstat",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
		Path:      "/etc/init.d/sysstat",
	})
}
