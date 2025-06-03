// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

// Ipv4Address structure that contains details about an IPv4
type Ipv4Address struct {
	IP        string
	Subnet    string
	CIDR      string
	Broadcast string
	Gateway   string
}

// NewIpv4WithMask generates a new Ipv4Address using the IP address and
// subnet mask. It calculates the subnet, CIDR and broadcast address.
func NewIpv4WithMask(ip, mask string) Ipv4Address {
	subnet := calculateSubnetFromIPAndMask(ip, mask)
	return NewIpv4WithSubnet(ip, subnet)
}

// NewIpv4WithSubnet generates a new Ipv4Address using the IP address and
// subnet in a CIDR format (e.g. 172.31.16.0/20). It calculates the CIDR
// and broadcast address.
func NewIpv4WithSubnet(ip, subnet string) Ipv4Address {
	address := Ipv4Address{
		IP:     ip,
		Subnet: subnet,
	}

	// Calculate the CIDR
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Debug().Err(err).Msg("Ipv4Address> invalid subnet")
		return address
	}
	ones, _ := network.Mask.Size()
	netIP := net.ParseIP(address.IP)
	if netIP != nil {
		address.CIDR = fmt.Sprintf("%s/%d", netIP.String(), ones)
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
		log.Debug().Msg("broadcastAddressFrom> invalid IPv4 address")
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

// calculateSubnetFromIPAndMask calculates the subnet network address from an IP and subnet mask
func calculateSubnetFromIPAndMask(ipStr, maskStr string) string {
	// Parse the IP address
	ip := net.ParseIP(ipStr).To4()
	if ip == nil {
		log.Debug().Msg("calculateSubnetFromIPAndMask> invalid IPv4 address")
		return ""
	}

	// Parse the subnet mask
	var b1, b2, b3, b4 int
	n, err := fmt.Sscanf(maskStr, "%d.%d.%d.%d", &b1, &b2, &b3, &b4)
	if err != nil || n != 4 {
		log.Debug().Msg("calculateSubnetFromIPAndMask> invalid subnet mask")
		return ""
	}
	mask := net.IPv4Mask(byte(b1), byte(b2), byte(b3), byte(b4))

	// Calculate the network address (bitwise AND)
	network := make(net.IP, len(ip))
	for i := range ip {
		network[i] = ip[i] & mask[i]
	}

	// Get CIDR notation from the mask
	ones, _ := mask.Size()
	if ones == 0 {
		log.Debug().Msg("calculateSubnetFromIPAndMask> invalid subnet mask CIDR")
		return ""
	}
	return fmt.Sprintf("%s/%d", network.String(), ones)
}
