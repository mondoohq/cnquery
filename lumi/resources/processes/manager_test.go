package processes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/processes"
	motor "go.mondoo.io/mondoo/motor/motoros"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := processes.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := processes.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 41, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := processes.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 41, len(mounts))
}

// func TestManagerWindows(t *testing.T) {
// 	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/windows.toml"})
// 	require.NoError(t, err)
// 	m, err := motor.New(mock)
// 	require.NoError(t, err)

// 	mm, err := processes.ResolveManager(m)
// 	require.NoError(t, err)
// 	mounts, err := mm.List()
// 	require.NoError(t, err)

// 	assert.Equal(t, 5, len(mounts))
// }
