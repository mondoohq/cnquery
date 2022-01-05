package kernel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 40, len(mounts))
}

func TestManagerCentos(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/centos7.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)

	info, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, "3.10.0-1127.19.1.el7.x86_64", info.Version)
	assert.Equal(t, map[string]string{"console": "ttyS0,38400n8", "crashkernel": "auto", "elevator": "noop", "ro": ""}, info.Arguments)
	assert.Equal(t, "/boot/vmlinuz-3.10.0-1127.19.1.el7.x86_64", info.Path)
	assert.Equal(t, "UUID=ff6cbb65-ccab-489c-91a5-61b9b09e4d49", info.Device)

	mods, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 16, len(mods))
}

func TestManagerAmazonLinux1(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amznlinux1.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)

	info, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, "4.14.193-113.317.amzn1.x86_64", info.Version)
	assert.Equal(t, map[string]string{"console": "ttyS0", "nvme_core.io_timeout": "4294967295", "selinux": "0"}, info.Arguments)
	assert.Equal(t, "", info.Path)
	assert.Equal(t, "LABEL=/", info.Device)

	mods, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 21, len(mods))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/osx.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)

	info, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, "19.4.0", info.Version)
	assert.Equal(t, map[string]string(nil), info.Arguments)
	assert.Equal(t, "", info.Path)
	assert.Equal(t, "", info.Device)

	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 33, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/freebsd12.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.Modules()
	require.NoError(t, err)

	assert.Equal(t, 4, len(mounts))
}
