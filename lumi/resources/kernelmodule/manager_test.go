package kernelmodule_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.mondoo.io/mondoo/lumi/resources/kernelmodule"
	motor "go.mondoo.io/mondoo/motor/motoros"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := kernelmodule.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 40, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := kernelmodule.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 33, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := kernelmodule.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 4, len(mounts))
}
