// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBroadcastAddressFrom(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected string
	}{
		{"Valid /24 subnet", "192.168.1.0/24", "192.168.1.255"},
		{"Valid /30 subnet", "192.168.1.0/30", "192.168.1.3"},
		{"Valid /16 subnet", "10.0.0.0/16", "10.0.255.255"},
		{"Valid /8 subnet", "172.0.0.0/8", "172.255.255.255"},
		{"Single IP /32", "192.168.1.1/32", "192.168.1.1"},
		{"Full range /0", "0.0.0.0/0", "255.255.255.255"},
		{"Invalid CIDR", "invalid", ""},
		{"IPv6 CIDR", "2001:db8::/32", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := broadcastAddressFrom(tt.cidr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateSubnetFromIPAndMask(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		mask     string
		expected string
	}{
		// Valid Cases
		{"Valid /24 subnet", "192.168.1.100", "255.255.255.0", "192.168.1.0/24"},
		{"Valid /16 subnet", "10.1.2.3", "255.255.0.0", "10.1.0.0/16"},
		{"Valid /30 subnet", "172.16.5.9", "255.255.255.252", "172.16.5.8/30"},
		{"Valid /8 subnet", "8.45.67.89", "255.0.0.0", "8.0.0.0/8"},
		{"Valid /32 subnet (single IP)", "192.168.1.1", "255.255.255.255", "192.168.1.1/32"},

		// Edge Cases (Invalid Inputs)
		{"Invalid IP address", "invalid", "255.255.255.0", ""},
		{"More invalid IP address", "300.168.1.10", "255.255.255.0", ""},
		{"Invalid subnet mask", "192.168.1.100", "invalid", ""},
		{"More invalid mask", "192.168.1.10", "999.999.999.999", ""},
		{"IPv6 address (not supported)", "2001:db8::1", "ffff:ffff:ffff:ffff::", ""},

		// Non-contiguous Mask (should fail or return error)
		{"Mismatched mask (non-contiguous)", "192.168.1.100", "255.255.0.255", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSubnetFromIPAndMask(tt.ip, tt.mask)
			assert.Equal(t, tt.expected, result)
		})
	}
}
