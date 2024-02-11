// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/services"
)

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "macos",
			Family: []string{"unix", "darwin"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(serviceList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "freebsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(serviceList))
}

func TestManagerDragonflybsd5(t *testing.T) {
	mock, err := mock.New(0, "./testdata/dragonfly5.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "dragonflybsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 11, len(serviceList))
}

func TestManagerOpenBsd6(t *testing.T) {
	mock, err := mock.New(0, "./testdata/openbsd6.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "openbsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 70, len(serviceList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.New(0, "./testdata/windows2019.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 1, len(serviceList))
}

func TestManagerUbuntu2204(t *testing.T) {
	mock, err := mock.New(0, "./testdata/ubuntu2204.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "ubuntu",
			Version: "22.04",
			Family:  []string{"ubuntu", "linux"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 264, len(serviceList))
}

func TestManagerPhoton(t *testing.T) {
	mock, err := mock.New(0, "./testdata/photon.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "photon",
			Version: "8.1.10",
			Family:  []string{"photon", "linux"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 138, len(serviceList))
}
