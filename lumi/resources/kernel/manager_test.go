package kernel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 40, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 33, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 4, len(mounts))
}
