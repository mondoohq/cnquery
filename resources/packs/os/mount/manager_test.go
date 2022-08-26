package mount_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources/packs/os/mount"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := mount.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/osx.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := mount.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 4, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/freebsd12.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := mount.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(mounts))
}
