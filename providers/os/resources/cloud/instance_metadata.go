// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog/log"
)

// InstanceMetadata is the data struct that the `OSCloud` interface uses
// to retrieve metadata from a cloud instance.
type InstanceMetadata struct {
	PublicHostname  string
	PrivateHostname string

	PublicIpv4  []Ipv4Address
	PrivateIpv4 []Ipv4Address

	Metadata any
}

// MqlID tries to generate a unique id for MQL resource
func (m InstanceMetadata) MqlID() string {
	switch {
	case m.PublicHostname != "":
		return fmt.Sprintf("cloud.instance/public/%s", m.PublicHostname)
	case m.PrivateHostname != "":
		return fmt.Sprintf("cloud.instance/private/%s", m.PrivateHostname)
	case m.PublicIP() != "":
		return fmt.Sprintf("cloud.instance/public/%s", m.PublicIP())
	case m.PrivateIP() != "":
		return fmt.Sprintf("cloud.instance/private/%s", m.PrivateIP())
	default:
		return "cloud.instance/unknown"
	}
}

// PublicIP returns the first public ip address found (used for defaults in `cloud.instance`)
func (m InstanceMetadata) PublicIP() string {
	for _, ip := range m.PublicIpv4 {
		if ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}

// PrivateIP returns the first private ip address found (used for defaults in `cloud.instance`)
func (m InstanceMetadata) PrivateIP() string {
	for _, ip := range m.PrivateIpv4 {
		if ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}

// AddOrUpdatePublicIP adds or updates one or many Ipv4Addresses
func (m *InstanceMetadata) AddOrUpdatePublicIP(ips ...Ipv4Address) {
	if m.PublicIpv4 == nil {
		m.PublicIpv4 = make([]Ipv4Address, 0)
	}

	for _, ip := range ips {
		index := m.findPublicIP(ip.IP)
		if index < 0 {
			// not found, add
			log.Trace().Str("ip", ip.IP).Msg("os.cloud.metadata> add public ip")
			m.PublicIpv4 = append(m.PublicIpv4, ip)
			continue
		}

		// found, update
		log.Trace().Str("ip", ip.IP).Msg("os.cloud.metadata> update public ip")
		m.mergePublicIP(index, ip)
	}
}
func (m *InstanceMetadata) mergePublicIP(index int, ip Ipv4Address) {
	merged := mergeIPs(m.PublicIpv4[index], ip)
	m.PublicIpv4[index] = merged
}
func (m *InstanceMetadata) findPublicIP(ip string) int {
	return slices.IndexFunc(m.PublicIpv4, func(address Ipv4Address) bool {
		return address.IP == ip
	})
}

// AddOrUpdatePrivateIP adds or updates one or many Ipv4Addresses
func (m *InstanceMetadata) AddOrUpdatePrivateIP(ips ...Ipv4Address) {
	if m.PrivateIpv4 == nil {
		m.PrivateIpv4 = make([]Ipv4Address, 0)
	}

	for _, ip := range ips {
		index := m.findPrivateIP(ip.IP)
		if index < 0 {
			// not found, add
			log.Trace().Str("ip", ip.IP).Msg("os.cloud.metadata> add private ip")
			m.PrivateIpv4 = append(m.PrivateIpv4, ip)
			continue
		}

		// found, update
		log.Trace().Str("ip", ip.IP).Msg("os.cloud.metadata> update private ip")
		m.mergePrivateIP(index, ip)
	}
}

func (m *InstanceMetadata) mergePrivateIP(index int, ip Ipv4Address) {
	merged := mergeIPs(m.PrivateIpv4[index], ip)
	m.PrivateIpv4[index] = merged
}
func (m *InstanceMetadata) findPrivateIP(ip string) int {
	return slices.IndexFunc(m.PrivateIpv4, func(address Ipv4Address) bool {
		return address.IP == ip
	})
}

// mergeIPs takes two Ipv4Address and merge them together. We give preference
// to the first ip provided.
func mergeIPs(ip1, ip2 Ipv4Address) Ipv4Address {
	if ip1.Subnet == "" {
		ip1.Subnet = ip2.Subnet
	}
	if ip1.Gateway == "" {
		ip1.Gateway = ip2.Gateway
	}
	if ip1.CIDR == "" {
		ip1.CIDR = ip2.CIDR
	}
	if ip1.Broadcast == "" {
		ip1.Broadcast = ip2.Broadcast
	}
	return ip1
}
