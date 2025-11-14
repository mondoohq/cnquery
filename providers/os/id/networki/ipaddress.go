// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// IPAddress structure that contains details about an IP address (v4 or v6)
type IPAddress struct {
	IP        net.IP `json:"ip"`
	Subnet    string `json:"subnet"`
	CIDR      string `json:"cidr"`
	Broadcast string `json:"broadcast"`
	Gateway   string `json:"gateway"`
}

// IPVersion represents either a version 4 or version 6 of an ip address.
type IPVersion string

const (
	IPv4 IPVersion = "IPv4"
	IPv6 IPVersion = "IPv6"
)

// Version returns the version of the ip address.
func (address IPAddress) Version() (IPVersion, bool) {
	if address.IP != nil {
		if address.IP.To4() != nil {
			return IPv4, true
		}
		if address.IP.To16() != nil {
			return IPv6, true
		}
	}
	return IPVersion(""), false
}

// NewIPAddress generates a new IPAddress and returns if it is valid or not
func NewIPAddress(ip string) (address IPAddress, ok bool) {
	address = IPAddress{
		IP: net.ParseIP(ip),
	}
	if address.IP != nil {
		ok = true
	}
	return
}

// NewIPv4WithMask generates a new IPAddress using the IP address and
// subnet mask. It calculates the subnet, CIDR and broadcast address.
func NewIPv4WithMask(ip, mask string) IPAddress {
	subnet := calculateSubnetFromIPv4AndMask(ip, mask)
	return NewIPWithSubnet(ip, subnet)
}

// NewIPWithPrefixLength generates a new IPAddress using the IP address and prefix length.
func NewIPWithPrefixLength(ip string, prefixLength int) (address IPAddress, ok bool) {
	if strings.Contains(ip, ":") {
		address = NewIPv6WithPrefixLength(ip, prefixLength)
	} else {
		address = NewIPv4WithPrefixLength(ip, prefixLength)
	}

	if address.IP != nil {
		ok = true
	}

	return
}

// NewIPv4WithPrefixLength generates a new IPAddress using the IP address and
// prefix length. It calculates the subnet, CIDR and broadcast address.
//
// NOTE: Use for IPv4 only.
func NewIPv4WithPrefixLength(ip string, prefixLength int) IPAddress {
	subnet := calculateSubnetFromIPv4AndPrefixLength(ip, prefixLength)
	return NewIPWithSubnet(ip, subnet)
}

// NewIPv6WithPrefixLength generates a new IPAddress using the IP address
// and prefix length. It calculates the subnet and CIDR format.
//
// NOTE: Use for IPv6 only.
func NewIPv6WithPrefixLength(ip string, prefixLength int) IPAddress {
	subnet := calculateSubnetFromIPv6AndPrefixLength(ip, prefixLength)
	return NewIPWithSubnet(ip, subnet)
}

// NewIPWithSubnet generates a new IPAddress using the IP address and
// subnet in a CIDR format (e.g. 172.31.16.0/20 or 2001:db8::/64).
// It calculates the CIDR and, only for IPv4, the broadcast address.
func NewIPWithSubnet(ip, subnet string) IPAddress {
	address := IPAddress{
		IP:     net.ParseIP(ip),
		Subnet: subnet,
	}

	// Calculate the CIDR
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Debug().Err(err).Msg("IPAddress> invalid subnet")
		return address
	}
	ones, _ := network.Mask.Size()
	if address.IP != nil {
		address.CIDR = fmt.Sprintf("%s/%d", address.IP.String(), ones)
	}

	// Calculate the broadcast address
	address.Broadcast = broadcastAddressFrom(address.CIDR)

	return address
}

// broadcastAddressFrom calculates the broadcast address for a given subnet
func broadcastAddressFrom(cidr string) string {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Debug().Err(err).Msg("broadcastAddressFrom> invalid CIDR")
		return ""
	}

	// Convert IP to a byte slice
	ip := ipNet.IP.To4()
	if ip == nil {
		// the IP address is v6, they don't have broadcast
		log.Debug().Str("CIDR", cidr).Msg("broadcastAddressFrom> IPv6 detected, skipping")
		return ""
	}

	// Calculate broadcast address: set all host bits to 1
	broadcast := make(net.IP, len(ip))
	copy(broadcast, ip)
	for i, maskByte := range ipNet.Mask {
		broadcast[i] |= ^maskByte
	}

	return broadcast.String()
}

// calculateSubnetFromIPAndMask calculates the subnet network address from an IPv4 and subnet mask
func calculateSubnetFromIPv4AndMask(ipv4Str, maskStr string) string {
	// Parse the IPv4 address
	ip := net.ParseIP(ipv4Str).To4()
	if ip == nil {
		log.Debug().
			Str("ipaddress", ipv4Str).
			Str("mask", maskStr).
			Msg("calculateSubnetFromIPv4AndMask> invalid IPv4 address")
		return ""
	}

	mask, err := parseIPv4Mask(maskStr)
	if err != nil {
		log.Debug().
			Str("ipaddress", ipv4Str).
			Str("mask", maskStr).
			Msg("calculateSubnetFromIPv4AndMask> invalid subnet mask")
		return ""
	}

	// Calculate the network address (bitwise AND)
	network := make(net.IP, len(ip))
	for i := range ip {
		network[i] = ip[i] & mask[i]
	}

	// Get CIDR notation from the mask
	ones, _ := mask.Size()
	if ones == 0 {
		log.Debug().
			Str("ipaddress", ipv4Str).
			Str("mask", maskStr).
			Msg("calculateSubnetFromIPv4AndMask> invalid subnet mask CIDR")
		return ""
	}
	return fmt.Sprintf("%s/%d", network.String(), ones)
}

// calculateSubnetFromIPv6AndPrefixLength calculates the subnet from an IPv6 address and prefix length.
func calculateSubnetFromIPv6AndPrefixLength(ipStr string, prefixLength int) string {
	// Parse the IPv6 address
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() != nil {
		log.Debug().Msg("calculateSubnetFromIPv6AndPrefixLength> invalid IPv6 address")
		return ""
	}

	// Convert IP to 16-byte format
	ip = ip.To16()

	// Create subnet mask from prefix length
	mask := net.CIDRMask(prefixLength, 128)

	// Calculate the network address (bitwise AND)
	network := make(net.IP, len(ip))
	for i := range ip {
		network[i] = ip[i] & mask[i]
	}

	// Return subnet in CIDR notation
	return fmt.Sprintf("%s/%d", network.String(), prefixLength)
}

// calculateSubnetFromIPv4AndPrefixLength calculates the subnet from an IPv4 address and prefix length.
func calculateSubnetFromIPv4AndPrefixLength(ipStr string, prefixLength int) string {
	// Parse the IPv4 address
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		fmt.Println("calculateSubnetFromIPv4AndPrefixLength> invalid IPv4 address")
		return ""
	}

	// Convert to IPv4 format
	ip = ip.To4()

	// Create subnet mask from prefix length
	mask := net.CIDRMask(prefixLength, 32)

	// Calculate the network address (bitwise AND)
	network := make(net.IP, len(ip))
	for i := range ip {
		network[i] = ip[i] & mask[i]
	}

	// Return subnet in CIDR notation
	return fmt.Sprintf("%s/%d", network.String(), prefixLength)
}

// parseIPv4Mask parses a subnet mask in either binary or hex formats
func parseIPv4Mask(maskStr string) (net.IPMask, error) {
	msgErr := "unable to parse mask"

	// Try parsing the subnet mask with format 255.255.255.255
	var b1, b2, b3, b4 int
	n, err := fmt.Sscanf(maskStr, "%d.%d.%d.%d", &b1, &b2, &b3, &b4)
	if err == nil && n == 4 {
		return net.IPv4Mask(byte(b1), byte(b2), byte(b3), byte(b4)), nil
	}

	// Try parsing the subnet mask with format 0xffffffff
	if !strings.HasPrefix(maskStr, "0x") || len(maskStr) != 10 {
		return nil, errors.New(msgErr)
	}

	// Convert hex string to a 32-bit integer
	maskInt, err := strconv.ParseUint(maskStr[2:], 16, 32)
	if err != nil {
		return nil, errors.New(msgErr)
	}

	// Convert to 4 bytes (big-endian)
	return net.IPv4Mask(
		byte((maskInt>>24)&0xFF),
		byte((maskInt>>16)&0xFF),
		byte((maskInt>>8)&0xFF),
		byte(maskInt&0xFF),
	), nil
}
