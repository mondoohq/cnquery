package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/resources/packs/os/services"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/centos6.toml")
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
	mock, err := mock.NewFromTomlFile("./testdata/ubuntu1404.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 9, len(serviceList))
}

func TestManagerOpensuse13(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.NewFromTomlFile("./testdata/opensuse13.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 31, len(serviceList))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/osx.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(serviceList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/freebsd12.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(serviceList))
}

func TestManagerDragonflybsd5(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/dragonfly5.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 11, len(serviceList))
}

func TestManagerOpenBsd6(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/openbsd6.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 70, len(serviceList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/windows2019.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := services.ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 1, len(serviceList))
}
