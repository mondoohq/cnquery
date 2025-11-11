// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"net"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

// Route represents a network route entry
type Route struct {
	Destination string
	Gateway     string
	Flags       []string
	Interface   string
	Platform    *inventory.Platform // Platform-specific flag handling
}

// IsDefaultRoute checks if a route is a default route (destination is 0.0.0.0/0 or ::/0)
func (r *Route) IsDefaultRoute() bool {
	return r.Destination == "0.0.0.0" || r.Destination == "0.0.0.0/0" ||
		r.Destination == "::" || r.Destination == "::/0" || r.Destination == "default"
}

// IsIPv4 checks if the route's gateway is an IPv4 address
func (r *Route) IsIPv4() bool {
	if r.Gateway == "" {
		return false
	}
	ip := net.ParseIP(r.Gateway)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6 checks if the route's gateway is an IPv6 address
func (r *Route) IsIPv6() bool {
	if r.Gateway == "" {
		return false
	}
	ip := net.ParseIP(r.Gateway)
	if ip == nil {
		return false
	}
	return ip.To4() == nil && ip.To16() != nil
}
