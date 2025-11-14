// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetMAC(t *testing.T) {
	iface := &Interface{}
	iface.SetMAC("00:1A:2B:3C:4D:5E")
	assert.Equal(t, "00:1A:2B:3C:4D:5E", iface.MACAddress)
	assert.Equal(t, "Ayecom Technology", iface.Vendor)
}

func TestAddOrUpdateInterfaces(t *testing.T) {
	iface1 := Interface{Name: "eth0", MACAddress: "00:1A:2B:3C:4D:5E"}
	iface2 := Interface{Name: "eth1", MACAddress: "00:1A:2B:3C:4D:5F"}
	result := AddOrUpdateInterfaces([]Interface{iface1}, []Interface{iface2})
	assert.Len(t, result, 2)

	// Test updating an existing interface
	iface3 := Interface{Name: "eth0", MTU: 1500}
	result = AddOrUpdateInterfaces(result, []Interface{iface3})
	assert.Len(t, result, 2)
	assert.Equal(t, 1500, result[0].MTU)
}

func TestMergeInterfaces(t *testing.T) {
	iface1 := Interface{Name: "eth0", MACAddress: ""}
	iface2 := Interface{Name: "eth0", MACAddress: "00:1A:2B:3C:4D:5E"}
	merged := mergeInterfaces(iface1, iface2)
	assert.Equal(t, "00:1A:2B:3C:4D:5E", merged.MACAddress)

	// Test merging flags
	iface1.Flags = []string{"UP"}
	iface2.Flags = []string{"BROADCAST"}
	merged = mergeInterfaces(iface1, iface2)
	assert.ElementsMatch(t, []string{"UP", "BROADCAST"}, merged.Flags)
}

func TestAddOrUpdateIP(t *testing.T) {
	iface := &Interface{Name: "eth0"}
	ip1 := IPAddress{IP: net.ParseIP("192.168.1.1")}
	iface.AddOrUpdateIP(ip1)
	assert.Len(t, iface.IPAddresses, 1)
	assert.Equal(t, "192.168.1.1", iface.IPAddresses[0].IP.String())

	// Test updating an existing IP
	ip2 := IPAddress{IP: net.ParseIP("192.168.1.1"), Subnet: "192.168.1.0/24"}
	iface.AddOrUpdateIP(ip2)
	assert.Len(t, iface.IPAddresses, 1)
	assert.Equal(t, "192.168.1.0/24", iface.IPAddresses[0].Subnet)

	// Test adding a new IP
	ip3 := IPAddress{IP: net.ParseIP("192.168.1.2")}
	iface.AddOrUpdateIP(ip3)
	assert.Len(t, iface.IPAddresses, 2)
}
