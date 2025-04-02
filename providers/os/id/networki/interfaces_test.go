// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki_test

import (
	"net"
	"testing"

	subject "go.mondoo.com/cnquery/v11/providers/os/id/networki"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
)

func TestInterfacesDarwin(t *testing.T) {
	conn, err := mock.New(0, "./testdata/macos.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	interfaces, err := subject.Interfaces(conn, platform)
	require.NoError(t, err)
	assert.Len(t, interfaces, 26)

	index := subject.FindInterface(interfaces, subject.Interface{Name: "en0"})
	if assert.NotEqual(t, -1, index) {
		en0 := interfaces[index]
		assert.Equal(t, "en0", en0.Name)
		assert.Equal(t, "80:a9:97:40:12:53", en0.MACAddress)
		assert.Equal(t, "Apple", en0.Vendor)
		assert.Equal(t, 1500, en0.MTU)
		if assert.NotNil(t, en0.Active) {
			assert.True(t, *en0.Active)
		}
		assert.Nil(t, en0.Virtual)
		assert.Equal(t, []string{"UP", "BROADCAST", "SMART", "RUNNING", "SIMPLEX", "MULTICAST"}, en0.Flags)
		if assert.NotEmpty(t, en0.IPAddresses) {
			i4 := en0.FindIP(net.ParseIP("192.168.86.36"))
			if assert.NotEqual(t, -1, i4) {
				ipv4 := en0.IPAddresses[i4]
				assert.Equal(t, "192.168.86.36", ipv4.IP.String())
				assert.Equal(t, "192.168.86.36/24", ipv4.CIDR)
				assert.Equal(t, "192.168.86.0/24", ipv4.Subnet)
				assert.Equal(t, "192.168.86.255", ipv4.Broadcast)
				assert.Equal(t, "192.168.86.1", ipv4.Gateway)
			}
			i6 := en0.FindIP(net.ParseIP("fd19:f27d:7e31:1af4:1cd0:9dc4:e6b0:ab13"))
			if assert.NotEqual(t, -1, i6) {
				ipv6 := en0.IPAddresses[i6]
				assert.Equal(t, "fd19:f27d:7e31:1af4:1cd0:9dc4:e6b0:ab13", ipv6.IP.String())
				assert.Equal(t, "fd19:f27d:7e31:1af4:1cd0:9dc4:e6b0:ab13/64", ipv6.CIDR)
				assert.Equal(t, "fd19:f27d:7e31:1af4::/64", ipv6.Subnet)
				assert.Equal(t, "", ipv6.Broadcast)
				assert.Equal(t, "", ipv6.Gateway)
			}
		}
	}
}

func TestInterfacesLinux(t *testing.T) {
	conn, err := mock.New(0, "./testdata/linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	interfaces, err := subject.Interfaces(conn, platform)
	require.NoError(t, err)
	assert.Len(t, interfaces, 2)

	index := subject.FindInterface(interfaces, subject.Interface{Name: "enX0"})
	if assert.NotEqual(t, -1, index) {
		enX0 := interfaces[index]
		assert.Equal(t, "enX0", enX0.Name)
		assert.Equal(t, "0a:ff:de:6b:e3:19", enX0.MACAddress)
		assert.Equal(t, "", enX0.Vendor)
		assert.Equal(t, 9001, enX0.MTU)
		if assert.NotNil(t, enX0.Active) {
			assert.True(t, *enX0.Active)
		}
		if assert.NotNil(t, enX0.Virtual) {
			assert.False(t, *enX0.Virtual)
		}
		assert.Equal(t, []string{"BROADCAST", "MULTICAST", "UP", "LOWER_UP"}, enX0.Flags)
		if assert.NotEmpty(t, enX0.IPAddresses) {
			i4 := enX0.FindIP(net.ParseIP("172.31.24.71"))
			if assert.NotEqual(t, -1, i4) {
				ipv4 := enX0.IPAddresses[i4]
				assert.Equal(t, "172.31.24.71", ipv4.IP.String())
				assert.Equal(t, "172.31.24.71/20", ipv4.CIDR)
				assert.Equal(t, "172.31.16.0/20", ipv4.Subnet)
				assert.Equal(t, "172.31.31.255", ipv4.Broadcast)
				assert.Equal(t, "172.31.16.1", ipv4.Gateway)
			}
			i6 := enX0.FindIP(net.ParseIP("fe80::8ff:deff:fe6b:e319"))
			if assert.NotEqual(t, -1, i6) {
				ipv6 := enX0.IPAddresses[i6]
				assert.Equal(t, "fe80::8ff:deff:fe6b:e319", ipv6.IP.String())
				assert.Equal(t, "fe80::8ff:deff:fe6b:e319/64", ipv6.CIDR)
				assert.Equal(t, "fe80::/64", ipv6.Subnet)
				assert.Equal(t, "", ipv6.Broadcast)
				assert.Equal(t, "", ipv6.Gateway)
			}
		}
	}
}

func TestInterfacesWindows(t *testing.T) {
	conn, err := mock.New(0, "./testdata/windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	interfaces, err := subject.Interfaces(conn, platform)
	require.NoError(t, err)
	assert.Len(t, interfaces, 4)

	index := subject.FindInterface(interfaces, subject.Interface{Name: "Ethernet0"})
	if assert.NotEqual(t, -1, index) {
		ethernet0 := interfaces[index]
		assert.Equal(t, "Ethernet0", ethernet0.Name)
		assert.Equal(t, "00-50-56-B0-9A-A5", ethernet0.MACAddress)
		assert.Equal(t, "", ethernet0.Vendor)
		assert.Equal(t, 1500, ethernet0.MTU)
		if assert.NotNil(t, ethernet0.Active) {
			assert.True(t, *ethernet0.Active)
		}
		if assert.NotNil(t, ethernet0.Virtual) {
			assert.False(t, *ethernet0.Virtual)

		}
		assert.Empty(t, ethernet0.Flags)
		if assert.NotEmpty(t, ethernet0.IPAddresses) {
			i4 := ethernet0.FindIP(net.ParseIP("192.168.5.38"))
			if assert.NotEqual(t, -1, i4) {
				ipv4 := ethernet0.IPAddresses[i4]
				assert.Equal(t, "192.168.5.38", ipv4.IP.String())
				assert.Equal(t, "192.168.5.38/24", ipv4.CIDR)
				assert.Equal(t, "192.168.5.0/24", ipv4.Subnet)
				assert.Equal(t, "192.168.5.255", ipv4.Broadcast)
				assert.Equal(t, "192.168.5.1", ipv4.Gateway)
			}
		}
	}

	index = subject.FindInterface(interfaces, subject.Interface{Name: "Teredo Tunneling Pseudo-Interface"})
	if assert.NotEqual(t, -1, index) {
		teredoTunneling := interfaces[index]
		if assert.NotEmpty(t, teredoTunneling.IPAddresses) {
			i6 := teredoTunneling.FindIP(net.ParseIP("2001:0:2851:782c:869:1f7d:a331:f3e1"))
			if assert.NotEqual(t, -1, i6) {
				ipv6 := teredoTunneling.IPAddresses[i6]
				assert.Equal(t, "2001:0:2851:782c:869:1f7d:a331:f3e1", ipv6.IP.String())
				assert.Equal(t, "2001:0:2851:782c:869:1f7d:a331:f3e1/64", ipv6.CIDR)
				assert.Equal(t, "2001:0:2851:782c::/64", ipv6.Subnet)
				assert.Equal(t, "", ipv6.Broadcast)
				assert.Equal(t, "::", ipv6.Gateway)
			}
		}
	}
}
