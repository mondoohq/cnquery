// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import "fmt"

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
