package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/services"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/linux_systemd.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 102, len(mounts))
}

func TestManagerAmzn1(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/amzn1.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 16, len(mounts))
}

func TestManagerCentos6(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/centos6.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(mounts))
}

func TestManagerUbuntu1404(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/ubuntu1404.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 9, len(mounts))
}

func TestManagerOpensuse13(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/opensuse13.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 31, len(mounts))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(mounts))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(mounts))
}

func TestManagerDragonflybsd5(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/dragonfly5.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 11, len(mounts))
}

func TestManagerOpenBsd6(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/openbsd6.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 70, len(mounts))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/windows2019.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 1, len(mounts))
}
