package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/resources/services"
)

func TestManagerDebian(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "debian",
			Family: []string{"debian", "linux"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 102, len(mounts))
}

func TestManagerAmzn1(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.New("./testdata/amzn1.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "amazonlinux",
			Family: []string{"linux"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 16, len(mounts))
}

func TestManagerCentos6(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.New("./testdata/centos6.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "centos",
			Family: []string{"linux", "redhat"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	mounts, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(mounts))
}

func TestManagerUbuntu1404(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.New("./testdata/ubuntu1404.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "ubuntu"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 9, len(serviceList))
}

func TestManagerOpensuse13(t *testing.T) {
	// tests fallback to upstart service
	mock, err := mock.New("./testdata/opensuse13.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "suse",
			Family: []string{"suse", "linux"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 31, len(serviceList))
}

func TestManagerMacos(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "macos",
			Family: []string{"unix", "darwin"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 15, len(serviceList))
}

func TestManagerFreebsd(t *testing.T) {
	mock, err := mock.New("./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "freebsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 25, len(serviceList))
}

func TestManagerDragonflybsd5(t *testing.T) {
	mock, err := mock.New("./testdata/dragonfly5.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "dragonflybsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 11, len(serviceList))
}

func TestManagerOpenBsd6(t *testing.T) {
	mock, err := mock.New("./testdata/openbsd6.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "openbsd",
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 70, len(serviceList))
}

func TestManagerWindows(t *testing.T) {
	mock, err := mock.New("./testdata/windows2019.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	})
	require.NoError(t, err)

	mm, err := services.ResolveManager(mock)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 1, len(serviceList))
}
