// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
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
