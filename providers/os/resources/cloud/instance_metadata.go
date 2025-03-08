// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import "fmt"

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

func (m InstanceMetadata) PublicIP() string {
	for _, ip := range m.PublicIpv4 {
		if ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}

func (m InstanceMetadata) PrivateIP() string {
	for _, ip := range m.PrivateIpv4 {
		if ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}
