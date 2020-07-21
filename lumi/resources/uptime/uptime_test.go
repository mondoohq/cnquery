package uptime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/uptime"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestUptimeOnLinux(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/linux.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	ut, err := uptime.New(m)
	require.NoError(t, err)

	required, err := ut.Duration()
	require.NoError(t, err)

	assert.Equal(t, "19m0s", required.String())
}

func TestUptimeOnFreebsd(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	ut, err := uptime.New(m)
	require.NoError(t, err)

	required, err := ut.Duration()
	require.NoError(t, err)

	assert.Equal(t, "24m0s", required.String())
}

func TestUptimeOnWindows(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/windows.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	ut, err := uptime.New(m)
	require.NoError(t, err)

	required, err := ut.Duration()
	require.NoError(t, err)

	assert.Equal(t, "3m45.8270365s", required.String())
}
