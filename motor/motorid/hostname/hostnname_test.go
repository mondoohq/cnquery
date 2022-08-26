package hostname_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/motorid/hostname"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestHostnameLinuxEtcHostname(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/hostname_arch.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	hostame, err := hostname.Hostname(provider, p)
	require.NoError(t, err)

	assert.Equal(t, "9be843c4be9f", hostame)
}

func TestHostnameLinux(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/hostname_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	hostame, err := hostname.Hostname(provider, p)
	require.NoError(t, err)

	assert.Equal(t, "abefed34cc9c", hostame)
}

func TestHostnameWindows(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/hostname_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	hostame, err := hostname.Hostname(provider, p)
	require.NoError(t, err)

	assert.Equal(t, "WIN-ABCDEFGVHLD", hostame)
}

func TestHostnameMacos(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/hostname_macos.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	hostame, err := hostname.Hostname(provider, p)
	require.NoError(t, err)

	assert.Equal(t, "moonshot.local", hostame)
}
