//go:build windows

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNetstatPowerShellOutput(t *testing.T) {
	// Sample netstat command output
	netstatJSON := `
	[
    {
        "P1":  "Active",
        "P2":  "Routes:",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "Network",
        "P2":  "Destination",
        "P3":  "Netmask",
        "P4":  "Gateway",
        "P5":  "Interface",
        "P6":  "Metric"
    },
    {
        "P1":  "",
        "P2":  "0.0.0.0",
        "P3":  "0.0.0.0",
        "P4":  "192.168.64.1",
        "P5":  "192.168.64.3",
        "P6":  15
    },
    {
        "P1":  "",
        "P2":  "127.0.0.0",
        "P3":  "255.0.0.0",
        "P4":  "On-link",
        "P5":  "127.0.0.1",
        "P6":  331
    },
    {
        "P1":  "",
        "P2":  "127.0.0.1",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "127.0.0.1",
        "P6":  331
    },
    {
        "P1":  "",
        "P2":  "127.255.255.255",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "127.0.0.1",
        "P6":  331
    },
    {
        "P1":  "",
        "P2":  "192.168.64.0",
        "P3":  "255.255.255.0",
        "P4":  "On-link",
        "P5":  "192.168.64.3",
        "P6":  271
    },
    {
        "P1":  "",
        "P2":  "192.168.64.3",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "192.168.64.3",
        "P6":  271
    },
    {
        "P1":  "",
        "P2":  "192.168.64.255",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "192.168.64.3",
        "P6":  271
    },
    {
        "P1":  "",
        "P2":  "224.0.0.0",
        "P3":  "240.0.0.0",
        "P4":  "On-link",
        "P5":  "127.0.0.1",
        "P6":  331
    },
    {
        "P1":  "",
        "P2":  "224.0.0.0",
        "P3":  "240.0.0.0",
        "P4":  "On-link",
        "P5":  "192.168.64.3",
        "P6":  271
    },
    {
        "P1":  "",
        "P2":  "255.255.255.255",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "127.0.0.1",
        "P6":  331
    },
    {
        "P1":  "",
        "P2":  "255.255.255.255",
        "P3":  "255.255.255.255",
        "P4":  "On-link",
        "P5":  "192.168.64.3",
        "P6":  271
    },
    {
        "P1":  "Persistent",
        "P2":  "Routes:",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "None",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "IPv6",
        "P2":  "Route",
        "P3":  "Table",
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "Active",
        "P2":  "Routes:",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "If",
        "P3":  "Metric",
        "P4":  "Network",
        "P5":  "Destination",
        "P6":  "Gateway"
    },
    {
        "P1":  "",
        "P2":  1,
        "P3":  331,
        "P4":  "::1/128",
        "P5":  "On-link",
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "fd28:df6a:5fc8:2cb::/64",
        "P5":  "On-link",
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "fd28:df6a:5fc8:2cb:cd60:a4f0:52ca:3c3/128",
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "On-link",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "fd28:df6a:5fc8:2cb:d33a:a509:e9d8:681a/128",
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "On-link",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "fe80::/64",
        "P5":  "On-link",
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "fe80::9b29:7b8f:bc2:f9bf/128",
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "On-link",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  1,
        "P3":  331,
        "P4":  "ff00::/8",
        "P5":  "On-link",
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  13,
        "P3":  271,
        "P4":  "ff00::/8",
        "P5":  "On-link",
        "p6":  null
    },
    {
        "P1":  "Persistent",
        "P2":  "Routes:",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    },
    {
        "P1":  "",
        "P2":  "None",
        "p3":  null,
        "p4":  null,
        "p5":  null,
        "p6":  null
    }
	]`

	// Expected routes with mocked interface name mapping
	expectedRoutes := []Route{
		{Destination: "0.0.0.0", Gateway: "192.168.64.1", Interface: "Ethernet"},
		{Destination: "127.0.0.0/8", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "127.0.0.1/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "127.255.255.255/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "192.168.64.0/24", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "192.168.64.3/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "192.168.64.255/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "224.0.0.0/4", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "224.0.0.0/4", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "255.255.255.255/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "255.255.255.255/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "::1/128", Gateway: "::", Interface: ""},
		{Destination: "fd28:df6a:5fc8:2cb::/64", Gateway: "::", Interface: ""},
		{Destination: "fd28:df6a:5fc8:2cb:cd60:a4f0:52ca:3c3/128", Gateway: "::", Interface: ""},
		{Destination: "fd28:df6a:5fc8:2cb:d33a:a509:e9d8:681a/128", Gateway: "::", Interface: ""},
		{Destination: "fe80::/64", Gateway: "::", Interface: ""},
		{Destination: "fe80::9b29:7b8f:bc2:f9bf/128", Gateway: "::", Interface: ""},
		{Destination: "ff00::/8", Gateway: "::", Interface: ""},
		{Destination: "ff00::/8", Gateway: "::", Interface: ""},
	}

	n := &neti{}

	// Mock IP to interface name mapping based on test data
	ipToNameMap := map[string]string{
		"127.0.0.1":    "Loopback Pseudo-Interface 1",
		"192.168.64.3": "Ethernet",
	}

	routes, err := n.parseNetstatPowerShellOutput(netstatJSON, ipToNameMap)
	require.NoError(t, err)

	// Compare routes (order may differ, so check by destination+gateway+interface)
	assert.Equal(t, len(expectedRoutes), len(routes), "route count mismatch")

	// Create maps for easier comparison
	expectedMap := make(map[string]Route)
	for _, r := range expectedRoutes {
		key := r.Destination + "|" + r.Gateway + "|" + r.Interface
		expectedMap[key] = r
	}

	actualMap := make(map[string]Route)
	for _, r := range routes {
		key := r.Destination + "|" + r.Gateway + "|" + r.Interface
		actualMap[key] = r
	}

	// Check all expected routes exist
	for key, expected := range expectedMap {
		actual, exists := actualMap[key]
		assert.True(t, exists, "missing route: %s", key)
		if exists {
			assert.Equal(t, expected.Destination, actual.Destination, "destination mismatch for %s", key)
			assert.Equal(t, expected.Gateway, actual.Gateway, "gateway mismatch for %s", key)
			assert.Equal(t, expected.Interface, actual.Interface, "interface mismatch for %s", key)
		}
	}
}

func TestParsePowerShellGetNetRouteOutput(t *testing.T) {
	powerShellJSON := `[
		{
			"DestinationPrefix":  "255.255.255.255/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "255.255.255.255/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "127.0.0.1"
		},
		{
			"DestinationPrefix":  "224.0.0.0/4",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "224.0.0.0/4",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "127.0.0.1"
		},
		{
			"DestinationPrefix":  "192.168.64.255/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "192.168.64.3/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "192.168.64.0/24",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "127.255.255.255/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "127.0.0.1"
		},
		{
			"DestinationPrefix":  "127.0.0.1/32",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "127.0.0.1"
		},
		{
			"DestinationPrefix":  "127.0.0.0/8",
			"NextHop":  "0.0.0.0",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  2,
			"InterfaceIP":  "127.0.0.1"
		},
		{
			"DestinationPrefix":  "0.0.0.0/0",
			"NextHop":  "192.168.64.1",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  0,
			"AddressFamily":  2,
			"InterfaceIP":  "192.168.64.3"
		},
		{
			"DestinationPrefix":  "ff00::/8",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "ff00::/8",
			"NextHop":  "::",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "::1"
		},
		{
			"DestinationPrefix":  "fe80::9b29:7b8f:bc2:f9bf/128",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "fe80::/64",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "fd28:df6a:5fc8:2cb:d33a:a509:e9d8:681a/128",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "fd28:df6a:5fc8:2cb:cd60:a4f0:52ca:3c3/128",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "fd28:df6a:5fc8:2cb::/64",
			"NextHop":  "::",
			"InterfaceIndex":  13,
			"InterfaceAlias":  "Ethernet",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "fe80::9b29:7b8f:bc2:f9bf%13"
		},
		{
			"DestinationPrefix":  "::1/128",
			"NextHop":  "::",
			"InterfaceIndex":  1,
			"InterfaceAlias":  "Loopback Pseudo-Interface 1",
			"RouteMetric":  256,
			"AddressFamily":  23,
			"InterfaceIP":  "::1"
		}
	]`

	expectedRoutes := []Route{
		// IPv4 routes
		{Destination: "0.0.0.0/0", Gateway: "192.168.64.1", Interface: "Ethernet"},
		{Destination: "127.0.0.0/8", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "127.0.0.1/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "127.255.255.255/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "192.168.64.0/24", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "192.168.64.3/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "192.168.64.255/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "224.0.0.0/4", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "224.0.0.0/4", Gateway: "192.168.64.3", Interface: "Ethernet"},
		{Destination: "255.255.255.255/32", Gateway: "127.0.0.1", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "255.255.255.255/32", Gateway: "192.168.64.3", Interface: "Ethernet"},
		// IPv6 routes
		{Destination: "::1/128", Gateway: "::", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "fd28:df6a:5fc8:2cb::/64", Gateway: "::", Interface: "Ethernet"},
		{Destination: "fd28:df6a:5fc8:2cb:cd60:a4f0:52ca:3c3/128", Gateway: "::", Interface: "Ethernet"},
		{Destination: "fd28:df6a:5fc8:2cb:d33a:a509:e9d8:681a/128", Gateway: "::", Interface: "Ethernet"},
		{Destination: "fe80::/64", Gateway: "::", Interface: "Ethernet"},
		{Destination: "fe80::9b29:7b8f:bc2:f9bf/128", Gateway: "::", Interface: "Ethernet"},
		{Destination: "ff00::/8", Gateway: "::", Interface: "Loopback Pseudo-Interface 1"},
		{Destination: "ff00::/8", Gateway: "::", Interface: "Ethernet"},
	}

	n := &neti{}
	routes, err := n.parsePowerShellGetNetRouteOutput(powerShellJSON)
	require.NoError(t, err)

	assert.Equal(t, len(expectedRoutes), len(routes), "route count mismatch")

	expectedMap := make(map[string]Route)
	for _, r := range expectedRoutes {
		key := r.Destination + "|" + r.Gateway + "|" + r.Interface
		expectedMap[key] = r
	}

	actualMap := make(map[string]Route)
	for _, r := range routes {
		key := r.Destination + "|" + r.Gateway + "|" + r.Interface
		actualMap[key] = r
	}

	// Check all expected routes exist
	for key, expected := range expectedMap {
		actual, exists := actualMap[key]
		assert.True(t, exists, "missing route: %s", key)
		if exists {
			assert.Equal(t, expected.Destination, actual.Destination, "destination mismatch for %s", key)
			assert.Equal(t, expected.Gateway, actual.Gateway, "gateway mismatch for %s", key)
			assert.Equal(t, expected.Interface, actual.Interface, "interface mismatch for %s", key)
		}
	}
}
