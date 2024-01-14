// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mount_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/mount"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{Family: []string{"linux"}},
	})
	require.NoError(t, err)

	mm, err := mount.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{Family: []string{"unix"}},
	})
	require.NoError(t, err)

	mm, err := mount.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 4, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New("./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{Family: []string{"unix"}},
	})
	require.NoError(t, err)

	mm, err := mount.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(mounts))
}
