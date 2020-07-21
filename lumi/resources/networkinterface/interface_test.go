package networkinterface_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/networkinterface"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

// TODO: compare go network interface results with mac/linux parser
// func TestLocaltNetworkInterface(t *testing.T) {
// 	trans, err := local.New()
// 	require.NoError(t, err)

// 	m, err := motor.New(trans)
// 	require.NoError(t, err)

// 	ifaces := networkinterface.New(m)
// 	list, err := ifaces.Interfaces()
// 	require.NoError(t, err)
// 	assert.True(t, len(list) > 0)
// 	inet := list[0]
// 	assert.Equal(t, "lo0", inet.Name)
// 	assert.Equal(t, 1, inet.Index)
// 	assert.Equal(t, 16384, inet.MTU)
// 	assert.Equal(t, "up|loopback|multicast", inet.Flags.String())
// 	assert.Equal(t, "", inet.HardwareAddr.String())
// 	assert.True(t, len(inet.Addrs) > 0)
// 	assert.True(t, len(inet.MulticastAddrs) > 0)

// 	inet = list[3]
// 	assert.Equal(t, "en0", inet.Name)
// 	assert.Equal(t, 5, inet.Index)
// 	assert.Equal(t, 1500, inet.MTU)
// 	assert.Equal(t, "up|broadcast|multicast", inet.Flags.String())
// 	assert.Equal(t, "8c:85:90:80:1b:e9", inet.HardwareAddr.String())
// 	assert.True(t, len(inet.Addrs) > 0)
// 	assert.True(t, len(inet.MulticastAddrs) > 0)

// 	// for i := range list {
// 	// 	log.Debug().Msg(list[i].Name)
// 	// }

// 	ip, err := networkinterface.HostIP(list)
// 	require.NoError(t, err)
// 	assert.Equal(t, "192.168.178.45", ip)
// }

func TestWindowsRemoteInterface(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/windows.toml"})
	require.NoError(t, err)

	m, err := motor.New(mock)
	require.NoError(t, err)

	ifaces := networkinterface.New(m)
	list, err := ifaces.Interfaces()
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	inet := list[0]
	assert.Equal(t, "Ethernet", inet.Name)
	assert.Equal(t, 6, inet.Index)
	assert.Equal(t, 0, inet.MTU)
	assert.Equal(t, "up|broadcast|multicast", inet.Flags.String())
	assert.Equal(t, "00:15:5d:f2:3b:1d", inet.HardwareAddr.String())

	assert.Equal(t, 2, len(inet.Addrs))
	assert.Equal(t, "fe80::ed94:1267:afb5:bb76", inet.Addrs[0].String())
	assert.Equal(t, "192.168.178.112", inet.Addrs[1].String())
	// the windows resourse does not support multicast adresses
	assert.True(t, len(inet.MulticastAddrs) == 0)

	ip, err := networkinterface.HostIP(list)
	require.NoError(t, err)
	assert.Equal(t, "192.168.178.112", ip)
}

func TestMacOsRegex(t *testing.T) {
	line := "lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384"

	m := networkinterface.IfconfigInterfaceLine.FindStringSubmatch(line)
	assert.Equal(t, "lo0", m[1])
	assert.Equal(t, "UP,LOOPBACK,RUNNING,MULTICAST", m[3])
	assert.Equal(t, "16384", m[4])
}

func TestMacOSRemoteInterface(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/macos.toml"})
	require.NoError(t, err)

	m, err := motor.New(mock)
	require.NoError(t, err)

	ifaces := networkinterface.New(m)
	list, err := ifaces.Interfaces()
	require.NoError(t, err)
	assert.Equal(t, 17, len(list))
	inet := list[0]
	assert.Equal(t, "lo0", inet.Name)
	assert.Equal(t, 1, inet.Index)
	assert.Equal(t, 16384, inet.MTU)
	assert.Equal(t, "up|loopback|multicast", inet.Flags.String())
	assert.Equal(t, "", inet.HardwareAddr.String())
	assert.True(t, len(inet.Addrs) > 0)
	assert.True(t, len(inet.MulticastAddrs) == 0)

	inetAdapter, err := ifaces.InterfaceByName("en0")
	require.NoError(t, err)
	assert.Equal(t, "en0", inetAdapter.Name)
	assert.Equal(t, "8c:85:90:80:1b:e9", inetAdapter.HardwareAddr.String())

	ip, err := networkinterface.HostIP(list)
	require.NoError(t, err)
	assert.Equal(t, "192.168.178.45", ip)
}

func TestLinuxRemoteInterface(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/linux_remote.toml"})
	require.NoError(t, err)

	m, err := motor.New(mock)
	require.NoError(t, err)

	ifaces := networkinterface.New(m)
	list, err := ifaces.Interfaces()
	require.NoError(t, err)
	assert.True(t, len(list) == 2)

	inet, err := ifaces.InterfaceByName("lo")
	assert.Equal(t, "lo", inet.Name)
	assert.Equal(t, 1, inet.Index)
	assert.Equal(t, 0, inet.MTU)
	assert.Equal(t, "up|loopback", inet.Flags.String())
	assert.Equal(t, "", inet.HardwareAddr.String())
	assert.True(t, len(inet.Addrs) == 2)
	assert.True(t, len(inet.MulticastAddrs) == 0)

	inet, err = ifaces.InterfaceByName("eth0")
	require.NoError(t, err)
	assert.Equal(t, "eth0", inet.Name)
	assert.Equal(t, 2, inet.Index)
	assert.Equal(t, 0, inet.MTU)
	assert.Equal(t, "up|broadcast", inet.Flags.String())
	assert.Equal(t, "", inet.HardwareAddr.String())
	assert.True(t, len(inet.Addrs) == 2)
	assert.True(t, len(inet.MulticastAddrs) == 0)

	ip, err := networkinterface.HostIP(list)
	require.NoError(t, err)
	assert.Equal(t, "10.128.0.4", ip)
}

func TestLinuxRemoteInterfaceFlannel(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/linux_flannel.toml"})
	require.NoError(t, err)

	m, err := motor.New(mock)
	require.NoError(t, err)

	ifaces := networkinterface.New(m)
	list, err := ifaces.Interfaces()
	require.NoError(t, err)
	assert.True(t, len(list) == 4)

	ip, err := networkinterface.HostIP(list)
	require.NoError(t, err)
	assert.Equal(t, "192.168.101.90", ip)
}
