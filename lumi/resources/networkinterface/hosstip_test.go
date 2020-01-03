package networkinterface_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/networkinterface"
	motor "go.mondoo.io/mondoo/motor/motoros"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestHostIp(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/macos.toml"})
	require.NoError(t, err)

	m, err := motor.New(mock)
	require.NoError(t, err)

	ifaces := networkinterface.New(m)
	interfaces, err := ifaces.Interfaces()

	require.NoError(t, err)
	assert.True(t, len(interfaces) > 0)

	ip, err := networkinterface.HostIP(interfaces)
	require.NoError(t, err)
	assert.Equal(t, "192.168.178.45", ip)
}
