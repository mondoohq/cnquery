// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/processes"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"linux", "unix"},
		},
	})
	require.NoError(t, err)

	mm, err := processes.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 3, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "darwin"},
		},
	})
	require.NoError(t, err)

	mm, err := processes.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 41, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "freebsd"},
		},
	})
	require.NoError(t, err)

	mm, err := processes.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 41, len(mounts))
}

// func TestManagerWindows(t *testing.T) {
//  mock, err := mock.New(0, "./testdata/windows.toml")
// 	require.NoError(t, err)
// 	m, err := motor.New(mock)
// 	require.NoError(t, err)

// 	mm, err := processes.ResolveManager(m)
// 	require.NoError(t, err)
// 	mounts, err := mm.List()
// 	require.NoError(t, err)

// 	assert.Equal(t, 5, len(mounts))
// }
