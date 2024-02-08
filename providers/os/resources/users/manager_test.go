// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/users"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "linux", "debian"},
		},
	})
	require.NoError(t, err)

	mm, err := users.ResolveManager(mock)
	require.NoError(t, err)
	userList, err := mm.List()
	require.NoError(t, err)

	usr := findUser(userList, "0")
	assert.Equal(t, "0", usr.ID)
	assert.Equal(t, int64(0), usr.Uid)
	assert.Equal(t, int64(0), usr.Gid)
	assert.Equal(t, "/root", usr.Home)
	assert.Equal(t, "root", usr.Name)
	assert.Equal(t, "/bin/bash", usr.Shell)

	assert.Equal(t, 13, len(userList))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"darwin"},
		},
	})
	require.NoError(t, err)

	mm, err := users.ResolveManager(mock)
	require.NoError(t, err)
	userList, err := mm.List()
	require.NoError(t, err)

	usr := findUser(userList, "0")
	assert.Equal(t, "0", usr.ID)
	assert.Equal(t, int64(0), usr.Uid)
	assert.Equal(t, int64(0), usr.Gid)
	assert.Equal(t, "/var/root /private/var/root", usr.Home)
	assert.Equal(t, "root", usr.Name)
	assert.Equal(t, "/bin/sh", usr.Shell)

	assert.Equal(t, 8, len(userList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix", "bsd"},
		},
	})
	require.NoError(t, err)

	mm, err := users.ResolveManager(mock)
	require.NoError(t, err)
	userList, err := mm.List()
	require.NoError(t, err)

	usr := findUser(userList, "0")
	assert.Equal(t, "0", usr.ID)
	assert.Equal(t, int64(0), usr.Uid)
	assert.Equal(t, int64(0), usr.Gid)
	assert.Equal(t, "/root", usr.Home)
	assert.Equal(t, "root", usr.Name)
	assert.Equal(t, "/bin/csh", usr.Shell)

	assert.Equal(t, 28, len(userList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.New(0, "./testdata/windows.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"windows"},
		},
	})
	require.NoError(t, err)

	mm, err := users.ResolveManager(mock)
	require.NoError(t, err)
	userList, err := mm.List()
	require.NoError(t, err)

	usr := findUser(userList, "S-1-5-21-2356735557-1575748656-448136971-500")
	assert.Equal(t, "S-1-5-21-2356735557-1575748656-448136971-500", usr.ID)
	assert.Equal(t, int64(-1), usr.Uid)
	assert.Equal(t, int64(-1), usr.Gid)
	assert.Equal(t, "", usr.Home)
	assert.Equal(t, "chris", usr.Name)
	assert.Equal(t, "", usr.Shell)

	assert.Equal(t, 5, len(userList))
}

func findUser(userList []*users.User, id string) *users.User {
	if len(userList) == 0 {
		return nil
	}

	for i := range userList {
		if userList[i].ID == id {
			return userList[i]
		}
	}
	return nil
}
