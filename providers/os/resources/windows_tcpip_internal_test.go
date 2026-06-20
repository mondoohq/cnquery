// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/registry"
)

// regItem builds a DWORD registry entry for tests.
func regItem(name string, number int64) registry.RegistryKeyItem {
	return registry.RegistryKeyItem{
		Key:   name,
		Value: registry.RegistryKeyValue{Number: number},
	}
}

func TestTcpipItemsToMap(t *testing.T) {
	t.Run("lower-cases value names and keeps numbers", func(t *testing.T) {
		got := tcpipItemsToMap([]registry.RegistryKeyItem{
			regItem("DisableIPSourceRouting", 2),
			regItem("EnableICMPRedirect", 0),
			regItem("KeepAliveTime", 300000),
		})
		assert.Equal(t, map[string]int64{
			"disableipsourcerouting": 2,
			"enableicmpredirect":     0,
			"keepalivetime":          300000,
		}, got)
	})

	t.Run("empty input yields empty map", func(t *testing.T) {
		got := tcpipItemsToMap(nil)
		assert.Equal(t, map[string]int64{}, got)
		assert.NotNil(t, got)
	})

	t.Run("preserves a lower-cased source name", func(t *testing.T) {
		// TcpMaxDataRetransmissions is stored lower-cased on some systems.
		got := tcpipItemsToMap([]registry.RegistryKeyItem{
			regItem("tcpmaxdataretransmissions", 3),
		})
		assert.Equal(t, map[string]int64{"tcpmaxdataretransmissions": 3}, got)
	})
}

func TestTcpipIntPtr(t *testing.T) {
	items := map[string]int64{
		"disableipsourcerouting":    2,
		"enableicmpredirect":        0,
		"tcpmaxdataretransmissions": 3,
	}

	t.Run("returns pointer to a present value", func(t *testing.T) {
		got := tcpipIntPtr(items, "DisableIPSourceRouting")
		require.NotNil(t, got)
		assert.Equal(t, int64(2), *got)
	})

	t.Run("distinguishes an explicit 0 from absent", func(t *testing.T) {
		// EnableICMPRedirect==0 is a real, compliant value and must not be
		// reported as null.
		got := tcpipIntPtr(items, "EnableICMPRedirect")
		require.NotNil(t, got)
		assert.Equal(t, int64(0), *got)
	})

	t.Run("returns nil for an absent value", func(t *testing.T) {
		got := tcpipIntPtr(items, "KeepAliveTime")
		assert.Nil(t, got)
	})

	t.Run("lookup is case-insensitive on the query name", func(t *testing.T) {
		// The MixedCase MQL name resolves the lower-cased stored value.
		got := tcpipIntPtr(items, "TcpMaxDataRetransmissions")
		require.NotNil(t, got)
		assert.Equal(t, int64(3), *got)
	})

	t.Run("nil on an empty map", func(t *testing.T) {
		assert.Nil(t, tcpipIntPtr(map[string]int64{}, "NodeType"))
	})

	t.Run("each returned pointer is independent", func(t *testing.T) {
		// guards against a range-variable aliasing bug where every pointer
		// would observe the last iterated value.
		a := tcpipIntPtr(items, "DisableIPSourceRouting")
		b := tcpipIntPtr(items, "TcpMaxDataRetransmissions")
		require.NotNil(t, a)
		require.NotNil(t, b)
		assert.Equal(t, int64(2), *a)
		assert.Equal(t, int64(3), *b)
	})
}

// TestTcpipCoverage documents the full set of CIS-required values backed by the
// resource and asserts the case-insensitive extraction works end-to-end for
// each one, including the lower-cased tcpmaxdataretransmissions casing.
func TestTcpipCoverage(t *testing.T) {
	// Tcpip\Parameters (IPv4) + lower-cased tcpmaxdataretransmissions
	ipv4 := tcpipItemsToMap([]registry.RegistryKeyItem{
		regItem("DisableIPSourceRouting", 2),
		regItem("EnableICMPRedirect", 0),
		regItem("KeepAliveTime", 300000),
		regItem("PerformRouterDiscovery", 0),
		regItem("tcpmaxdataretransmissions", 3),
	})
	for name, want := range map[string]int64{
		"DisableIPSourceRouting":    2,
		"EnableICMPRedirect":        0,
		"KeepAliveTime":             300000,
		"PerformRouterDiscovery":    0,
		"TcpMaxDataRetransmissions": 3,
	} {
		got := tcpipIntPtr(ipv4, name)
		require.NotNilf(t, got, "expected %s to be present", name)
		assert.Equalf(t, want, *got, "value for %s", name)
	}

	// Tcpip6\Parameters (IPv6)
	ipv6 := tcpipItemsToMap([]registry.RegistryKeyItem{
		regItem("DisableIPSourceRouting", 2),
	})
	got := tcpipIntPtr(ipv6, "DisableIPSourceRouting")
	require.NotNil(t, got)
	assert.Equal(t, int64(2), *got)

	// NetBT\Parameters (NetBIOS)
	netbios := tcpipItemsToMap([]registry.RegistryKeyItem{
		regItem("NodeType", 2),
		regItem("NoNameReleaseOnDemand", 1),
	})
	for name, want := range map[string]int64{
		"NodeType":              2,
		"NoNameReleaseOnDemand": 1,
	} {
		got := tcpipIntPtr(netbios, name)
		require.NotNilf(t, got, "expected %s to be present", name)
		assert.Equalf(t, want, *got, "value for %s", name)
	}
}
