// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v12/providers/os/resources/services"
)

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "macos",
			Family: []string{"unix", "darwin"},
		},
	}, mock.WithPath("./testdata/osx.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(serviceList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "freebsd",
			Family: []string{"unix"},
		},
	}, mock.WithPath("./testdata/freebsd12.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(serviceList))
}

func TestManagerDragonflybsd5(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "dragonflybsd",
			Family: []string{"unix"},
		},
	}, mock.WithPath("./testdata/dragonfly5.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 11, len(serviceList))
}

func TestManagerOpenBsd6(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "openbsd",
			Family: []string{"unix"},
		},
	}, mock.WithPath("./testdata/openbsd6.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 70, len(serviceList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	}, mock.WithPath("./testdata/windows2019.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 1, len(serviceList))
}

func TestManagerUbuntu2204(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "ubuntu",
			Version: "22.04",
			Family:  []string{"ubuntu", "linux"},
		},
	}, mock.WithPath("./testdata/ubuntu2204.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 264, len(serviceList))
}

func TestManagerPhoton(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "photon",
			Version: "8.1.10",
			Family:  []string{"photon", "linux"},
		},
	}, mock.WithPath("./testdata/photon.toml"))
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 138, len(serviceList))
}
