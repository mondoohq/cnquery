// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"errors"
	"net"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// Route represents a network route entry
type Route struct {
	Destination string
	Gateway     string
	Flags       int64
	Interface   string
}

// Routes returns the network routes of the system.
func Routes(conn shared.Connection, pf *inventory.Platform) ([]Route, error) {
	n := &neti{conn, pf}

	if pf.IsFamily(inventory.FAMILY_LINUX) {
		return n.detectLinuxRoutes()
	}
	if pf.IsFamily(inventory.FAMILY_DARWIN) {
		return n.detectDarwinRoutes()
	}
	if pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return n.detectWindowsRoutes()
	}

	return nil, errors.New("your platform is not supported for the detection of network routes")
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
