// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package networki

import (
	"encoding/json"
	"net"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// ipRouteJSON represents a route entry from 'ip -json route show table all'
type ipRouteJSON struct {
	Dst      string   `json:"dst"`
	Gateway  string   `json:"gateway,omitempty"`
	Dev      string   `json:"dev"`
	Type     string   `json:"type,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
	Table    string   `json:"table,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	Prefsrc  string   `json:"prefsrc,omitempty"`
	Metric   int      `json:"metric,omitempty"`
	Pref     string   `json:"pref,omitempty"`
	Flags    []string `json:"flags,omitempty"`
}

// Routes returns the network routes of the system.
func Routes(conn shared.Connection, pf *inventory.Platform) ([]Route, error) {
	n := &neti{conn, pf}

	if pf.IsFamily(inventory.FAMILY_LINUX) {
		return n.detectLinuxRoutes()
	}

	return nil, errors.New("your platform is not supported for the detection of network routes")
}

// detectLinuxRoutes detects network routes on Linux using 'ip -json route show table all'
func (n *neti) detectLinuxRoutes() ([]Route, error) {
	output, err := n.RunCommand("ip -json route show table all")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via ip command")
	}

	return n.parseIpRouteJSON(output)
}

// parseIpRouteJSON parses JSON output from 'ip -json route show table all'
func (n *neti) parseIpRouteJSON(output string) ([]Route, error) {
	var jsonRoutes []ipRouteJSON
	if err := json.Unmarshal([]byte(output), &jsonRoutes); err != nil {
		return nil, errors.Wrap(err, "failed to parse ip route JSON output")
	}

	var routes []Route
	for _, jsonRoute := range jsonRoutes {
		route := n.convertJSONRouteToRoute(jsonRoute)
		if route != nil {
			routes = append(routes, *route)
		}
	}

	return routes, nil
}

// convertJSONRouteToRoute converts an ipRouteJSON to a Route
func (n *neti) convertJSONRouteToRoute(jsonRoute ipRouteJSON) *Route {
	// Skip routes without a device
	if jsonRoute.Dev == "" {
		return nil
	}

	route := &Route{
		Interface: jsonRoute.Dev,
		Gateway:   jsonRoute.Gateway,
	}

	dest := jsonRoute.Dst
	if dest == "default" {
		var family string
		if ip := net.ParseIP(jsonRoute.Gateway); ip != nil {
			if ip.To4() != nil {
				family = "v4"
			} else {
				family = "v6"
			}
		} else if ip := net.ParseIP(jsonRoute.Prefsrc); ip != nil {
			if ip.To4() != nil {
				family = "v4"
			} else {
				family = "v6"
			}
		}

		if family == "v6" {
			dest = "::"
		} else {
			dest = "0.0.0.0"
		}
	}
	route.Destination = dest
	route.Flags = jsonRoute.Flags

	return route
}
