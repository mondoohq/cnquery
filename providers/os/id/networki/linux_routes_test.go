//go:build linux

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseIpRouteJSON(t *testing.T) {
	// Example JSON output from 'ip -json route show table all'
	jsonOutput := `
[
  {
    "dst": "default",
    "gateway": "192.168.178.1",
    "dev": "wlo1",
    "protocol": "dhcp",
    "prefsrc": "192.168.178.79",
    "metric": 600,
    "flags": []
  },
  {
    "dst": "172.17.0.0/16",
    "dev": "docker0",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.17.0.1",
    "flags": [
      "linkdown"
    ]
  },
  {
    "dst": "172.18.0.0/16",
    "dev": "br-01ca4fc8136f",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.18.0.1",
    "flags": []
  },
  {
    "dst": "172.19.0.0/16",
    "dev": "br-573cc74a8612",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.19.0.1",
    "flags": []
  },
  {
    "dst": "192.168.178.0/24",
    "dev": "wlo1",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "192.168.178.79",
    "metric": 600,
    "flags": []
  },
  {
    "type": "local",
    "dst": "127.0.0.0/8",
    "dev": "lo",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "127.0.0.1",
    "flags": []
  },
  {
    "type": "local",
    "dst": "127.0.0.1",
    "dev": "lo",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "127.0.0.1",
    "flags": []
  },
  {
    "type": "broadcast",
    "dst": "127.255.255.255",
    "dev": "lo",
    "table": "local",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "127.0.0.1",
    "flags": []
  },
  {
    "type": "local",
    "dst": "172.17.0.1",
    "dev": "docker0",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "172.17.0.1",
    "flags": []
  },
  {
    "type": "broadcast",
    "dst": "172.17.255.255",
    "dev": "docker0",
    "table": "local",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.17.0.1",
    "flags": [
      "linkdown"
    ]
  },
  {
    "type": "local",
    "dst": "172.18.0.1",
    "dev": "br-01ca4fc8136f",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "172.18.0.1",
    "flags": []
  },
  {
    "type": "broadcast",
    "dst": "172.18.255.255",
    "dev": "br-01ca4fc8136f",
    "table": "local",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.18.0.1",
    "flags": []
  },
  {
    "type": "local",
    "dst": "172.19.0.1",
    "dev": "br-573cc74a8612",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "172.19.0.1",
    "flags": []
  },
  {
    "type": "broadcast",
    "dst": "172.19.255.255",
    "dev": "br-573cc74a8612",
    "table": "local",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "172.19.0.1",
    "flags": []
  },
  {
    "type": "local",
    "dst": "192.168.178.79",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "scope": "host",
    "prefsrc": "192.168.178.79",
    "flags": []
  },
  {
    "type": "broadcast",
    "dst": "192.168.178.255",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "scope": "link",
    "prefsrc": "192.168.178.79",
    "flags": []
  },
  {
    "dst": "2a02:810a:983:f300::/64",
    "dev": "wlo1",
    "protocol": "ra",
    "metric": 600,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "2a02:810a:983:f300::/64",
    "gateway": "fe80::b2f2:8ff:fe4c:9c41",
    "dev": "wlo1",
    "protocol": "ra",
    "metric": 605,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fdad:a22e:9f09::/64",
    "dev": "wlo1",
    "protocol": "ra",
    "metric": 600,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fdad:a22e:9f09::/64",
    "gateway": "fe80::b2f2:8ff:fe4c:9c41",
    "dev": "wlo1",
    "protocol": "ra",
    "metric": 605,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "br-01ca4fc8136f",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "docker0",
    "protocol": "kernel",
    "metric": 256,
    "flags": [
      "linkdown"
    ],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "br-573cc74a8612",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "veth13dad49",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "vethce143da",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "vethc4f5eec",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "veth303a5c2",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "veth61d447b",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "fe80::/64",
    "dev": "wlo1",
    "protocol": "kernel",
    "metric": 1024,
    "flags": [],
    "pref": "medium"
  },
  {
    "dst": "default",
    "gateway": "fe80::b2f2:8ff:fe4c:9c41",
    "dev": "wlo1",
    "protocol": "ra",
    "metric": 600,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "::1",
    "dev": "lo",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:6b3f:c2b8:fe22:ac4c",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:7189:6322:2080:e9b3",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:7612:ec99:1a8e:b466",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:9a97:890e:dc47:d10f",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:ac76:8fd7:cda1:f9f7",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:b867:36f6:5030:3608",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:f696:40b7:26b9:63fb",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "2a02:810a:983:f300:ff5d:3e49:c4d2:3951",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:170:4bcd:7b4c:db31",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:10b6:5ffa:990c:6461",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:647f:a22b:646c:8d26",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:8148:77e7:3263:11b8",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:93cf:4ed1:1afa:bd41",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:e096:7de1:c750:bddd",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:e155:9c13:9c44:307",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fdad:a22e:9f09:0:f21d:ed26:bb49:dec6",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::42:1fff:feda:c5e0",
    "dev": "br-573cc74a8612",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::42:36ff:fe7d:ca5",
    "dev": "br-01ca4fc8136f",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::42:74ff:fe55:e6ae",
    "dev": "docker0",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::2019:9cff:fe49:50a1",
    "dev": "vethc4f5eec",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::5082:1eff:fed5:990a",
    "dev": "veth61d447b",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::6843:81ff:fe73:9c89",
    "dev": "veth13dad49",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::ac61:28ff:fef4:319a",
    "dev": "vethce143da",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::d43e:f5ff:fe24:dc2c",
    "dev": "veth303a5c2",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "local",
    "dst": "fe80::e0e1:af0:8971:6498",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 0,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "br-01ca4fc8136f",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "docker0",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [
      "linkdown"
    ],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "br-573cc74a8612",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "veth13dad49",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "vethce143da",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "vethc4f5eec",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "veth303a5c2",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "veth61d447b",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  },
  {
    "type": "multicast",
    "dst": "ff00::/8",
    "dev": "wlo1",
    "table": "local",
    "protocol": "kernel",
    "metric": 256,
    "flags": [],
    "pref": "medium"
  }
]
`

	n := &neti{}
	routes, err := n.parseIpRouteJSON(jsonOutput)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	assert.Greater(t, len(routes), 60, "Should parse all routes from JSON")

	tests := []struct {
		name          string
		destination   string
		gateway       string
		interfaceName string
		expectedFlags []string
	}{
		{
			name:          "IPv4 default route",
			destination:   "0.0.0.0",
			gateway:       "192.168.178.1",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "Docker network route with linkdown",
			destination:   "172.17.0.0/16",
			gateway:       "",
			interfaceName: "docker0",
			expectedFlags: []string{"linkdown"},
		},
		{
			name:          "Bridge network route",
			destination:   "172.18.0.0/16",
			gateway:       "",
			interfaceName: "br-01ca4fc8136f",
			expectedFlags: []string{},
		},
		{
			name:          "Another bridge network route",
			destination:   "172.19.0.0/16",
			gateway:       "",
			interfaceName: "br-573cc74a8612",
			expectedFlags: []string{},
		},
		{
			name:          "Local network route",
			destination:   "192.168.178.0/24",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "Loopback network route",
			destination:   "127.0.0.0/8",
			gateway:       "",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
		{
			name:          "Loopback host route",
			destination:   "127.0.0.1",
			gateway:       "",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
		{
			name:          "Loopback broadcast route",
			destination:   "127.255.255.255",
			gateway:       "",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
		{
			name:          "Docker bridge local IP",
			destination:   "172.17.0.1",
			gateway:       "",
			interfaceName: "docker0",
			expectedFlags: []string{},
		},
		{
			name:          "Docker bridge broadcast with linkdown",
			destination:   "172.17.255.255",
			gateway:       "",
			interfaceName: "docker0",
			expectedFlags: []string{"linkdown"},
		},
		{
			name:          "Bridge local IP",
			destination:   "172.18.0.1",
			gateway:       "",
			interfaceName: "br-01ca4fc8136f",
			expectedFlags: []string{},
		},
		{
			name:          "Bridge broadcast",
			destination:   "172.18.255.255",
			gateway:       "",
			interfaceName: "br-01ca4fc8136f",
			expectedFlags: []string{},
		},
		{
			name:          "Another bridge local IP",
			destination:   "172.19.0.1",
			gateway:       "",
			interfaceName: "br-573cc74a8612",
			expectedFlags: []string{},
		},
		{
			name:          "Another bridge broadcast",
			destination:   "172.19.255.255",
			gateway:       "",
			interfaceName: "br-573cc74a8612",
			expectedFlags: []string{},
		},
		{
			name:          "WLAN local IP",
			destination:   "192.168.178.79",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "WLAN broadcast",
			destination:   "192.168.178.255",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 network route",
			destination:   "2a02:810a:983:f300::/64",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 network route with gateway",
			destination:   "2a02:810a:983:f300::/64",
			gateway:       "fe80::b2f2:8ff:fe4c:9c41",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 ULA network route",
			destination:   "fdad:a22e:9f09::/64",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 ULA network route with gateway",
			destination:   "fdad:a22e:9f09::/64",
			gateway:       "fe80::b2f2:8ff:fe4c:9c41",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on bridge",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "br-01ca4fc8136f",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on docker with linkdown",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "docker0",
			expectedFlags: []string{"linkdown"},
		},
		{
			name:          "IPv6 link-local route on another bridge",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "br-573cc74a8612",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on veth13dad49",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "veth13dad49",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on vethce143da",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "vethce143da",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on vethc4f5eec",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "vethc4f5eec",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on veth303a5c2",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "veth303a5c2",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on veth61d447b",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "veth61d447b",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 link-local route on WLAN",
			destination:   "fe80::/64",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 default route",
			destination:   "::",
			gateway:       "fe80::b2f2:8ff:fe4c:9c41",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 localhost",
			destination:   "::1",
			gateway:       "",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on bridge",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "br-01ca4fc8136f",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on docker with linkdown",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "docker0",
			expectedFlags: []string{"linkdown"},
		},
		{
			name:          "IPv6 multicast route on another bridge",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "br-573cc74a8612",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on veth13dad49",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "veth13dad49",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on vethce143da",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "vethce143da",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on vethc4f5eec",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "vethc4f5eec",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route 4",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "veth303a5c2",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on fifth veth61d447b",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "veth61d447b",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 multicast route on WLAN",
			destination:   "ff00::/8",
			gateway:       "",
			interfaceName: "wlo1",
			expectedFlags: []string{},
		},
	}

	routeMap := make(map[string]*Route, len(routes))
	for i := range routes {
		key := routes[i].Destination + "|" + routes[i].Gateway + "|" + routes[i].Interface
		routeMap[key] = &routes[i]
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.destination + "|" + tt.gateway + "|" + tt.interfaceName
			found, exists := routeMap[key]

			require.True(t, exists, "Route not found: %s", tt.name)
			assert.Equal(t, tt.expectedFlags, found.Flags, "Flags should match for %s", tt.name)
		})
	}
}

func Test_parseLinuxRoutesFromProc(t *testing.T) {
	// Test data from Alpine container /proc/net/route
	procOutput := `Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT                                                       
eth0	00000000	010011AC	0003	0	0	0	00000000	0	0	0                                                                               
eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0                                                                               

`

	n := &neti{}
	routes, err := n.parseLinuxRoutesFromProc(procOutput)
	require.NoError(t, err)
	require.Len(t, routes, 2, "Should parse 2 routes from /proc/net/route")

	// Build route map for easy lookup
	routeMap := make(map[string]*Route, len(routes))
	for i := range routes {
		key := routes[i].Destination + "|" + routes[i].Gateway + "|" + routes[i].Interface
		routeMap[key] = &routes[i]
	}

	tests := []struct {
		name          string
		destination   string
		gateway       string
		interfaceName string
		expectedFlags []string
	}{
		{
			name:          "Default route",
			destination:   "0.0.0.0/0",
			gateway:       "172.17.0.1",
			interfaceName: "eth0",
			expectedFlags: []string{"UP"},
		},
		{
			name:          "Network route",
			destination:   "172.17.0.0/16",
			gateway:       "",
			interfaceName: "eth0",
			expectedFlags: []string{"UP"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.destination + "|" + tt.gateway + "|" + tt.interfaceName
			found, exists := routeMap[key]

			require.True(t, exists, "Route not found: %s", key)
			assert.Equal(t, tt.destination, found.Destination, "Destination should match")
			assert.Equal(t, tt.gateway, found.Gateway, "Gateway should match")
			assert.Equal(t, tt.interfaceName, found.Interface, "Interface should match")
			assert.Equal(t, tt.expectedFlags, found.Flags, "Flags should match")
		})
	}
}

func Test_parseLinuxIPv6RoutesFromProc(t *testing.T) {
	// Test data from Alpine container /proc/net/ipv6_route (exact content from alpine-proc-test/proc_net_ipv6_route.txt)
	procOutput := `00000000000000000000000000000000 00 00000000000000000000000000000000 00 00000000000000000000000000000000 ffffffff 00000001 00000000 00200200       lo
00000000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000002 00000000 80200001       lo
00000000000000000000000000000000 00 00000000000000000000000000000000 00 00000000000000000000000000000000 ffffffff 00000001 00000000 00200200       lo

`

	n := &neti{}
	routes, err := n.parseLinuxIPv6RoutesFromProc(procOutput)
	require.NoError(t, err)
	// The test data has 3 lines: 2 default routes (::/0) and 1 localhost route (::1/128)
	// So we expect 3 routes, but 2 are duplicates in the map
	require.GreaterOrEqual(t, len(routes), 2, "Should parse at least 2 routes from /proc/net/ipv6_route")

	// Build route map for easy lookup
	routeMap := make(map[string]*Route, len(routes))
	for i := range routes {
		key := routes[i].Destination + "|" + routes[i].Gateway + "|" + routes[i].Interface
		routeMap[key] = &routes[i]
	}

	tests := []struct {
		name          string
		destination   string
		gateway       string
		interfaceName string
		expectedFlags []string
	}{
		{
			name:          "IPv6 default route",
			destination:   "::/0",
			gateway:       "::",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
		{
			name:          "IPv6 localhost route",
			destination:   "::1/128",
			gateway:       "::",
			interfaceName: "lo",
			expectedFlags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.destination + "|" + tt.gateway + "|" + tt.interfaceName
			found, exists := routeMap[key]

			require.True(t, exists, "Route not found: %s", key)
			assert.Equal(t, tt.destination, found.Destination, "Destination should match")
			assert.Equal(t, tt.gateway, found.Gateway, "Gateway should match")
			assert.Equal(t, tt.interfaceName, found.Interface, "Interface should match")
			assert.Equal(t, tt.expectedFlags, found.Flags, "Flags should match")
		})
	}
}
