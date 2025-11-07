//go:build windows

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package networki

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/powershell"
)

const (
	// Windows AddressFamily constants from Get-NetRoute
	addressFamilyIPv6 = 23 // AF_INET6
)

// Routes returns the network routes of the system.
func Routes(conn shared.Connection, pf *inventory.Platform) ([]Route, error) {
	n := &neti{conn, pf}

	if pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return n.detectWindowsRoutes()
	}

	return nil, errors.New("your platform is not supported for the detection of network routes")
}

// detectWindowsRoutes detects network routes on Windows
// Tries 2 approaches in order: 1) PowerShell Get-NetRoute, 2) netstat -rn
func (n *neti) detectWindowsRoutes() ([]Route, error) {
	// Approach 1: PowerShell Get-NetRoute
	routes, err := n.detectWindowsRoutesViaPowerShell()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Msg("PowerShell Get-NetRoute failed, trying netstat")

	// Approach 2: netstat -rn with PowerShell ConvertFrom-String
	return n.detectWindowsRoutesViaNetstat()
}

// detectWindowsRoutesViaPowerShell uses PowerShell Get-NetRoute command
func (n *neti) detectWindowsRoutesViaPowerShell() ([]Route, error) {
	cmd := `Get-NetRoute | ForEach-Object {
		$route = $_
		$ifIndex = $route.InterfaceIndex
		$ifIP = (Get-NetIPAddress -InterfaceIndex $ifIndex -AddressFamily $route.AddressFamily -ErrorAction SilentlyContinue | Select-Object -First 1).IPAddress
		[PSCustomObject]@{
			DestinationPrefix = $route.DestinationPrefix
			NextHop = $route.NextHop
			InterfaceIndex = $route.InterfaceIndex
			InterfaceAlias = $route.InterfaceAlias
			RouteMetric = $route.RouteMetric
			AddressFamily = $route.AddressFamily
			InterfaceIP = $ifIP
		}
	} | ConvertTo-Json`
	command := powershell.Encode(cmd)

	output, err := n.RunCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via PowerShell Get-NetRoute")
	}

	return n.parsePowerShellGetNetRouteOutput(output)
}

// WindowsNetRoute represents a route from Get-NetRoute PowerShell command
type WindowsNetRoute struct {
	DestinationPrefix string `json:"DestinationPrefix"`
	NextHop           string `json:"NextHop"`
	InterfaceIndex    int    `json:"InterfaceIndex"`
	InterfaceAlias    string `json:"InterfaceAlias"`
	RouteMetric       int    `json:"RouteMetric"`
	AddressFamily     int    `json:"AddressFamily"` // addressFamilyIPv4 (2) = IPv4, addressFamilyIPv6 (23) = IPv6
	InterfaceIP       string `json:"InterfaceIP"`   // IP address of the interface
}

// parsePowerShellGetNetRouteOutput parses JSON output from Get-NetRoute PowerShell command
func (n *neti) parsePowerShellGetNetRouteOutput(output string) ([]Route, error) {
	var routes []Route

	// PowerShell may return a single object or an array
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
		// Skip routes without destination
		destination := netRoute.DestinationPrefix
		if destination == "" {
			continue
		}

		// Normalize gateway - osquery shows "::" for IPv6 local routes
		gateway := netRoute.NextHop
		if gateway == "" || gateway == "0.0.0.0" {
			// Empty or 0.0.0.0 gateway means on-link route
			if netRoute.AddressFamily == addressFamilyIPv6 {
				gateway = "::"
			} else {
				// For IPv4, use interface IP if available
				if netRoute.InterfaceIP != "" {
					gateway = netRoute.InterfaceIP
				} else {
					gateway = "0.0.0.0"
				}
			}
		}

		// osquery shows IP address for IPv4, empty for IPv6
		var iface string
		if netRoute.AddressFamily == addressFamilyIPv6 {
			iface = ""
		} else {
			iface = netRoute.InterfaceIP
			if iface == "" {
				iface = netRoute.InterfaceAlias
			}
			if iface == "" {
				iface = fmt.Sprintf("%d", netRoute.InterfaceIndex)
			}
		}

		routes = append(routes, Route{
			Destination: destination,
			Gateway:     gateway,
			Flags:       0,
			Interface:   iface,
		})
	}

	return routes, nil
}

// detectWindowsRoutesViaNetstat uses netstat -rn with PowerShell ConvertFrom-String
// Uses: $a = netstat -rn; $a[8..$a.count] | ConvertFrom-String | select p1,p2,p3,p4,p5,p6
func (n *neti) detectWindowsRoutesViaNetstat() ([]Route, error) {
	cmd := `$a = netstat -rn; $a[8..$a.count] | ConvertFrom-String | select p1,p2,p3,p4,p5,p6 | ConvertTo-Json`
	command := powershell.Encode(cmd)

	output, err := n.RunCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via netstat")
	}

	return n.parseNetstatPowerShellOutput(output)
}

// NetstatRoute represents a route parsed from netstat -rn via ConvertFrom-String
// IPv4 format: P1=empty, P2=Destination, P3=Netmask, P4=Gateway, P5=Interface, P6=Metric
// IPv6 format: P1=empty, P2=If, P3=Metric, P4=Network Destination, P5=Gateway, P6=Gateway
type NetstatRoute struct {
	P1 string `json:"p1"` // Empty (leading spaces) or header text
	P2 string `json:"p2"` // Destination (IPv4) or If (IPv6)
	P3 string `json:"p3"` // Netmask (IPv4) or Metric (IPv6)
	P4 string `json:"p4"` // Gateway (IPv4) or Network Destination (IPv6)
	P5 string `json:"p5"` // Interface (IPv4) or Gateway (IPv6)
	P6 string `json:"p6"` // Metric (IPv4) or Gateway (IPv6, when "On-link" appears)
}

// parseNetstatPowerShellOutput parses JSON output from netstat -rn via ConvertFrom-String
func (n *neti) parseNetstatPowerShellOutput(output string) ([]Route, error) {
	var routes []Route

	// PowerShell may return a single object or an array
	var netstatRoutes []NetstatRoute
	err := json.Unmarshal([]byte(output), &netstatRoutes)
	if err != nil {
		// If it fails, try parsing as a single object
		var singleRoute NetstatRoute
		err = json.Unmarshal([]byte(output), &singleRoute)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse netstat output")
		}
		netstatRoutes = []NetstatRoute{singleRoute}
	}

	var inIPv4Table, inIPv6Table bool
	var pendingIPv6Route *Route

	for i, route := range netstatRoutes {
		if n.isHeaderRow(route) {
			if strings.Contains(route.P1, "IPv6") || strings.Contains(route.P2, "IPv6") {
				inIPv6Table = true
				inIPv4Table = false
			} else if strings.Contains(route.P2, "Network") && strings.Contains(route.P3, "Destination") {
				inIPv4Table = true
				inIPv6Table = false
			} else if strings.Contains(route.P2, "If") && strings.Contains(route.P3, "Metric") {
				inIPv6Table = true
				inIPv4Table = false
			}
			continue
		}

		// Skip empty rows
		if route.P1 == "" && route.P2 == "" && route.P3 == "" && route.P4 == "" {
			continue
		}

		if inIPv4Table && !inIPv6Table {
			if route.P2 != "" && strings.Contains(route.P2, ".") {
				r := n.parseIPv4NetstatRoute(route)
				if r != nil {
					routes = append(routes, *r)
				}
			}
			continue
		}

		// Parse IPv6 routes: P2=If, P3=Metric, P4=Destination, P5=Gateway
		// Note: Sometimes "On-link" appears in P2 of next row, so we need to handle that
		if inIPv6Table {
			// Check if this row has "On-link" in P2 (it's a continuation of previous route)
			if strings.TrimSpace(route.P2) == "On-link" && pendingIPv6Route != nil {
				// Complete the pending route with "On-link" -> "::"
				pendingIPv6Route.Gateway = "::"
				routes = append(routes, *pendingIPv6Route)
				pendingIPv6Route = nil
				continue
			}

			// Check if P4 contains IPv6 address (destination)
			if route.P4 != "" && (strings.Contains(route.P4, ":") || strings.Contains(route.P4, "::")) {
				r := n.parseIPv6NetstatRoute(route)
				if r != nil {
					// If gateway is empty or "On-link" appears in next row, save for later
					if r.Gateway == "" {
						// Check next row for "On-link"
						if i+1 < len(netstatRoutes) && strings.TrimSpace(netstatRoutes[i+1].P2) == "On-link" {
							pendingIPv6Route = r
							continue
						}
						r.Gateway = "::"
					}
					routes = append(routes, *r)
				}
			}
		}
	}

	// Handle any pending IPv6 route
	if pendingIPv6Route != nil {
		pendingIPv6Route.Gateway = "::"
		routes = append(routes, *pendingIPv6Route)
	}

	return routes, nil
}

// isHeaderRow checks if a row is a header that should be skipped
func (n *neti) isHeaderRow(route NetstatRoute) bool {
	p1 := strings.TrimSpace(route.P1)
	p2 := strings.TrimSpace(route.P2)
	p3 := strings.TrimSpace(route.P3)

	// Check for header combinations
	if (p1 == "Network" && p2 == "Destination") ||
		(p2 == "If" && p3 == "Metric") ||
		(p1 == "Active" && p2 == "Routes:") ||
		(p1 == "Persistent" && p2 == "Routes:") ||
		(p1 == "IPv6" && p2 == "Route" && p3 == "Table") {
		return true
	}

	return false
}

// parseIPv4NetstatRoute parses an IPv4 route from netstat ConvertFrom-String output
// Format: P2=Destination, P3=Netmask, P4=Gateway, P5=Interface
func (n *neti) parseIPv4NetstatRoute(route NetstatRoute) *Route {
	if route.P2 == "" || route.P3 == "" {
		return nil
	}

	destination := strings.TrimSpace(route.P2)
	netmask := strings.TrimSpace(route.P3)
	gateway := strings.TrimSpace(route.P4)
	iface := strings.TrimSpace(route.P5)

	// Handle "On-link" gateway
	if gateway == "On-link" {
		gateway = iface
	}

	// Convert netmask to CIDR
	destIP := net.ParseIP(destination)
	if destIP == nil {
		return nil
	}

	maskIP := net.ParseIP(netmask)
	if maskIP == nil {
		return nil
	}

	mask := net.IPMask(maskIP.To4())
	if mask == nil {
		return nil
	}

	ones, bits := mask.Size()
	if bits != 32 {
		return nil
	}

	// Format destination with CIDR
	var dest string
	if destIP.Equal(net.IPv4zero) {
		dest = "0.0.0.0"
	} else {
		dest = fmt.Sprintf("%s/%d", destination, ones)
	}

	return &Route{
		Destination: dest,
		Gateway:     gateway,
		Flags:       0,
		Interface:   iface,
	}
}

// parseIPv6NetstatRoute parses an IPv6 route from netstat ConvertFrom-String output
// Format: P2=If, P3=Metric, P4=Network Destination, P5=Gateway
func (n *neti) parseIPv6NetstatRoute(route NetstatRoute) *Route {
	destination := strings.TrimSpace(route.P4)
	if destination == "" {
		return nil
	}

	gateway := strings.TrimSpace(route.P5)

	// Handle "On-link" gateway
	if gateway == "On-link" {
		gateway = "::"
	}

	// Normalize destination format
	dest := destination
	if !strings.Contains(dest, "/") {
		// Try to parse as IP and add /128 for host routes
		if ip := net.ParseIP(dest); ip != nil {
			if ip.Equal(net.IPv6zero) {
				dest = "::"
			} else {
				dest = fmt.Sprintf("%s/128", dest)
			}
		}
	}

	return &Route{
		Destination: dest,
		Gateway:     gateway,
		Flags:       0,
		Interface:   "", // Interface not available in netstat IPv6 output
	}
}
