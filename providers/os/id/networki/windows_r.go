// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/providers/os/resources/powershell"
)

// WindowsNetRoute represents a route from Get-NetRoute PowerShell command
type WindowsNetRoute struct {
	DestinationPrefix string `json:"DestinationPrefix"`
	NextHop           string `json:"NextHop"`
	InterfaceIndex    int    `json:"InterfaceIndex"`
	InterfaceAlias    string `json:"InterfaceAlias"`
	RouteMetric       int    `json:"RouteMetric"`
	AddressFamily     int    `json:"AddressFamily"` // 2 = IPv4, 23 = IPv6
}

// detectWindowsRoutes detects network routes on Windows using PowerShell Get-NetRoute
func (n *neti) detectWindowsRoutes() ([]Route, error) {
	// Use Get-NetRoute PowerShell command to get all routes
	cmd := `Get-NetRoute | Select-Object DestinationPrefix, NextHop, InterfaceIndex, InterfaceAlias, RouteMetric, AddressFamily | ConvertTo-Json`
	command := powershell.Encode(cmd)

	output, err := n.RunCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via Get-NetRoute")
	}

	return n.parseWindowsGetNetRouteOutput(output)
}

// parseWindowsGetNetRouteOutput parses JSON output from Get-NetRoute PowerShell command
func (n *neti) parseWindowsGetNetRouteOutput(output string) ([]Route, error) {
	var routes []Route

	// PowerShell may return a single object or an array
	// Try to parse as array first
	var netRoutes []WindowsNetRoute
	err := json.Unmarshal([]byte(output), &netRoutes)
	if err != nil {
		// If it fails, try parsing as a single object
		var singleRoute WindowsNetRoute
		err2 := json.Unmarshal([]byte(output), &singleRoute)
		if err2 != nil {
			return nil, errors.Wrap(err, "failed to parse Get-NetRoute output")
		}
		netRoutes = []WindowsNetRoute{singleRoute}
	}

	for _, netRoute := range netRoutes {
		// Skip routes without a gateway
		if netRoute.NextHop == "" || netRoute.NextHop == "0.0.0.0" {
			continue
		}

		iface := netRoute.InterfaceAlias
		if iface == "" {
			iface = fmt.Sprintf("%d", netRoute.InterfaceIndex)
		}

		// Set flags
		flags := int64(0)
		if netRoute.DestinationPrefix == "0.0.0.0/0" || netRoute.DestinationPrefix == "::/0" {
			flags |= 0x1 // Default route
		}

		// Normalize destination
		destination := netRoute.DestinationPrefix
		if destination == "" {
			continue
		}

		routes = append(routes, Route{
			Destination: destination,
			Gateway:     netRoute.NextHop,
			Flags:       flags,
			Interface:   iface,
		})
	}

	return routes, nil
}
