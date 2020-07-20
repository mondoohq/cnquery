package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/users"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := users.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := users.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := users.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/windows.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := users.ResolveManager(m)
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
