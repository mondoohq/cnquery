// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v9/providers/os/resources/services"
)

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", &inventory.Asset{
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
	mock, err := mock.New("./testdata/freebsd12.toml", &inventory.Asset{
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
	mock, err := mock.New("./testdata/dragonfly5.toml", &inventory.Asset{
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
	mock, err := mock.New("./testdata/openbsd6.toml", &inventory.Asset{
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
	mock, err := mock.New("./testdata/windows2019.toml", &inventory.Asset{
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

// TODO: these tests need to be reworked(new testdata is needed)
// we changed from using `systemctl list-units` to `systemctl list-unit-files`
//
//	func TestManagerDebian(t *testing.T) {
//		mock, err := mock.New("./testdata/ubuntu.toml", &inventory.Asset{
//			Platform: &inventory.Platform{
//				Name:    "ubuntu",
//				Version: "22.04",
//				Family:  []string{"ubuntu", "linux"},
//			},
//		})
//		require.NoError(t, err)
//
//		mm, err := services.ResolveManager(mock)
//		require.NoError(t, err)
//		mounts, err := mm.List()
//		require.NoError(t, err)
//
//		assert.Equal(t, 264, len(mounts))
//	}
//
//	func TestManagerAmzn1(t *testing.T) {
//		// tests fallback to upstart service
//		mock, err := mock.New("./testdata/amzn1.toml", &inventory.Asset{
//			Platform: &inventory.Platform{
//				Name:    "amazonlinux",
//				Version: "2018.03",
//				Family:  []string{"linux"},
//			},
//		})
//		require.NoError(t, err)
//
//		mm, err := services.ResolveManager(mock)
//		require.NoError(t, err)
//		mounts, err := mm.List()
//		require.NoError(t, err)
//
//		assert.Equal(t, 16, len(mounts))
//	}
//
//
//	func TestManagerUbuntu1404(t *testing.T) {
//		// tests fallback to upstart service
//		mock, err := mock.New("./testdata/ubuntu1404.toml", &inventory.Asset{
//			Platform: &inventory.Platform{
//				Name:    "ubuntu",
//				Version: "14.04",
//				Family:  []string{"linux", "ubuntu"},
//			},
//		})
//		require.NoError(t, err)
//
//		mm, err := services.ResolveManager(mock)
//		require.NoError(t, err)
//		serviceList, err := mm.List()
//		require.NoError(t, err)
//
//		assert.Equal(t, 9, len(serviceList))
//	}
//
//	func TestManagerOpensuse13(t *testing.T) {
//		// tests fallback to upstart service
//		mock, err := mock.New("./testdata/opensuse13.toml", &inventory.Asset{
//			Platform: &inventory.Platform{
//				Name:    "suse",
//				Version: "13",
//				Family:  []string{"suse", "linux"},
//			},
//		})
//		require.NoError(t, err)
//
//		mm, err := services.ResolveManager(mock)
//		require.NoError(t, err)
//		serviceList, err := mm.List()
//		require.NoError(t, err)
//
//		assert.Equal(t, 31, len(serviceList))
//	}
