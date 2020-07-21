package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/groups"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/debian.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := groups.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := groups.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := groups.ResolveManager(m)
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
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/windows.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := groups.ResolveManager(m)
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
