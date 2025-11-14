// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIPAddress(t *testing.T) {
	// Valid IPv4
	ip, ok := NewIPAddress("192.168.1.1")
	assert.True(t, ok)
	assert.Equal(t, "192.168.1.1", ip.IP.String())

	// Valid IPv6
	ip, ok = NewIPAddress("2001:db8::1")
	assert.True(t, ok)
	assert.Equal(t, "2001:db8::1", ip.IP.String())

	// Invalid IP
	ip, ok = NewIPAddress("invalid-ip")
	assert.False(t, ok)
}

func TestVersion(t *testing.T) {
	// IPv4 Address
	ip, _ := NewIPAddress("192.168.1.1")
	ver, valid := ip.Version()
	assert.True(t, valid)
	assert.Equal(t, IPv4, ver)

	// IPv6 Address
	ip, _ = NewIPAddress("2001:db8::1")
	ver, valid = ip.Version()
	assert.True(t, valid)
	assert.Equal(t, IPv6, ver)
}

func TestNewIPv4WithMask(t *testing.T) {
	address := NewIPv4WithMask("192.168.1.1", "255.255.255.0")
	assert.Equal(t, "192.168.1.0/24", address.Subnet)
	assert.Equal(t, "192.168.1.255", address.Broadcast)
}

func TestNewIPWithPrefixLength(t *testing.T) {
	address, ok := NewIPWithPrefixLength("192.168.1.1", 24)
	assert.True(t, ok)
	assert.Equal(t, "192.168.1.0/24", address.Subnet)
}

func TestNewIPv6WithPrefixLength(t *testing.T) {
	address := NewIPv6WithPrefixLength("2001:db8::1", 64)
	assert.Equal(t, "2001:db8::/64", address.Subnet)
}

func TestBroadcastAddressFrom(t *testing.T) {
	broadcast := broadcastAddressFrom("192.168.1.0/24")
	assert.Equal(t, "192.168.1.255", broadcast)

	broadcast = broadcastAddressFrom("2001:db8::/64")
	assert.Equal(t, "", broadcast) // IPv6 has no broadcast
}

func TestCalculateSubnetFromIPv4AndMask(t *testing.T) {
	subnet := calculateSubnetFromIPv4AndMask("192.168.1.10", "255.255.255.0")
	assert.Equal(t, "192.168.1.0/24", subnet)
}

func TestParseIPv4Mask(t *testing.T) {
	mask, err := parseIPv4Mask("255.255.255.0")
	assert.NoError(t, err)
	assert.Equal(t, net.IPv4Mask(255, 255, 255, 0), mask)

	mask, err = parseIPv4Mask("0xffffff00")
	assert.NoError(t, err)
	assert.Equal(t, net.IPv4Mask(255, 255, 255, 0), mask)

	_, err = parseIPv4Mask("invalid-mask")
	assert.Error(t, err)
}
