// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/groups"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "linux", "debian"},
		},
	})
	require.NoError(t, err)

	mm, err := groups.ResolveManager(mock)
	require.NoError(t, err)
	groupList, err := mm.List()
	require.NoError(t, err)

	grp := findGroup(groupList, "0")
	assert.Equal(t, "0", grp.ID)
	assert.Equal(t, int64(0), grp.Gid)
	assert.Equal(t, "root", grp.Name)
	assert.Equal(t, []string{}, grp.Members)

	assert.Equal(t, 23, len(groupList))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"darwin"},
		},
	})
	require.NoError(t, err)

	mm, err := groups.ResolveManager(mock)
	require.NoError(t, err)
	groupList, err := mm.List()
	require.NoError(t, err)

	grp := findGroup(groupList, "216")
	assert.Equal(t, "216", grp.ID)
	assert.Equal(t, int64(216), grp.Gid)
	assert.Equal(t, "_postgres", grp.Name)
	assert.Equal(t, []string{"_devicemgr", "_calendar", "_teamsserver", "_xserverdocs"}, grp.Members)

	assert.Equal(t, 3, len(groupList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New("./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "bsd"},
		},
	})
	require.NoError(t, err)

	mm, err := groups.ResolveManager(mock)
	require.NoError(t, err)
	groupList, err := mm.List()
	require.NoError(t, err)

	grp := findGroup(groupList, "0")
	assert.Equal(t, "0", grp.ID)
	assert.Equal(t, int64(0), grp.Gid)
	assert.Equal(t, "wheel", grp.Name)
	assert.Equal(t, []string{"root", "vagrant"}, grp.Members)

	assert.Equal(t, 36, len(groupList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.New("./testdata/windows.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"windows"},
		},
	})
	require.NoError(t, err)

	mm, err := groups.ResolveManager(mock)
	require.NoError(t, err)
	groupList, err := mm.List()
	require.NoError(t, err)

	grp := findGroup(groupList, "S-1-5-32-544")
	assert.Equal(t, "S-1-5-32-544", grp.ID)
	assert.Equal(t, int64(-1), grp.Gid)
	assert.Equal(t, "Administrators", grp.Name)
	assert.Equal(t, []string{}, grp.Members)

	assert.Equal(t, 25, len(groupList))
}

func findGroup(groupList []*groups.Group, id string) *groups.Group {
	if len(groupList) == 0 {
		return nil
	}

	for i := range groupList {
		if groupList[i].ID == id {
			return groupList[i]
		}
	}
	return nil
}
