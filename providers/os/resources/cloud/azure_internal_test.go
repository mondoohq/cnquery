// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureLoadbalancerIPAddressDetails_PublicIP(t *testing.T) {
	ipDetails := AzureLoadbalancerIPAddressDetails{FrontendIPAddress: "203.0.113.1"}
	ip, exists := ipDetails.PublicIP()
	assert.True(t, exists)
	assert.Equal(t, "203.0.113.1", ip.IP)
}

func TestAzureLoadbalancerIPAddressDetails_PrivateIP(t *testing.T) {
	ipDetails := AzureLoadbalancerIPAddressDetails{PrivateIPAddress: "10.0.0.1"}
	ip, exists := ipDetails.PrivateIP()
	assert.True(t, exists)
	assert.Equal(t, "10.0.0.1", ip.IP)
}

func TestAzureSubnet_CIDR(t *testing.T) {
	subnet := AzureSubnet{Address: "192.168.1.0", Prefix: "24"}
	assert.Equal(t, "192.168.1.0/24", subnet.CIDR())
}

func TestAzureNetworkInterfaceIpv4_PublicIPs(t *testing.T) {
	interfaceIpv4 := AzureNetworkInterfaceIpv4{
		IPAddress: []AzureIPAddress{
			{PublicIPAddress: "203.0.113.1"},
			{PublicIPAddress: ""},
		},
	}
	ips, exists := interfaceIpv4.PublicIPs()
	assert.True(t, exists)
	assert.Len(t, ips, 1)
	assert.Equal(t, "203.0.113.1", ips[0].IP)
}

func TestAzureNetworkInterfaceIpv4_PrivateIPs(t *testing.T) {
	interfaceIpv4 := AzureNetworkInterfaceIpv4{
		IPAddress: []AzureIPAddress{
			{PrivateIPAddress: "10.0.0.1"},
			{PrivateIPAddress: ""},
		},
	}
	ips, exists := interfaceIpv4.PrivateIPs()
	assert.True(t, exists)
	assert.Len(t, ips, 1)
	assert.Equal(t, "10.0.0.1", ips[0].IP)
}

func TestAzureNetworkInterfaceIpv4_findMatchingSubnet(t *testing.T) {
	interfaceIpv4 := AzureNetworkInterfaceIpv4{
		Subnet: []AzureSubnet{
			{Address: "192.168.1.0", Prefix: "24"},
			{Address: "10.0.0.0", Prefix: "16"},
		},
	}
	tests := []struct {
		ip       string
		exists   bool
		expected AzureSubnet
	}{
		{"192.168.1.100", true, AzureSubnet{Address: "192.168.1.0", Prefix: "24"}},
		{"10.0.1.50", true, AzureSubnet{Address: "10.0.0.0", Prefix: "16"}},
		{"8.8.8.8", false, AzureSubnet{}},
	}

	for _, test := range tests {
		netIP := net.ParseIP(test.ip)
		result, found := interfaceIpv4.findMatchingSubnet(netIP)
		assert.Equal(t, test.exists, found)
		if found {
			assert.Equal(t, test.expected, result)
		}
	}
}
