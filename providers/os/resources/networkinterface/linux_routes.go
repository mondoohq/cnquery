// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package networkinterface

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
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

// detectLinuxRoutes detects network routes on Linux
// First tries 'ip -json route show table all' (modern approach)
// Falls back to /proc/net/route and /proc/net/ipv6_route if ip -json is not available (e.g., Alpine Linux)
func (n *netr) detectLinuxRoutes() ([]Route, error) {
	output, err := n.RunCommand("ip -json route show table all")
	if err == nil {
		routes, err := n.parseIpRouteJSON(output)
		if err == nil {
			return routes, nil
		}
		log.Debug().Err(err).Msg("Failed to parse ip route JSON output, falling back to /proc/net/route and /proc/net/ipv6_route")
	}

	// ip -json failed (e.g., not available on Alpine), fall back to /proc
	// Get IPv4 routes from /proc/net/route
	ipv4Data, err := afero.ReadFile(n.connection.FileSystem(), "/proc/net/route")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read /proc/net/route")
	}

	ipv4Routes, err := n.parseLinuxRoutesFromProc(string(ipv4Data))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse IPv4 routes from /proc/net/route")
	}

	ipv6Data, err := afero.ReadFile(n.connection.FileSystem(), "/proc/net/ipv6_route")
	var ipv6Routes []Route
	if err == nil {
		ipv6Routes, err = n.parseLinuxIPv6RoutesFromProc(string(ipv6Data))
		if err != nil {
			// Some alpine systems may not have IPv6 enabled
			log.Debug().Err(err).Msg("Failed to parse IPv6 routes from /proc/net/ipv6_route")
			ipv6Routes = []Route{}
		}
	}

	return append(ipv4Routes, ipv6Routes...), nil
}

// parseIpRouteJSON parses JSON output from 'ip -json route show table all'
func (n *netr) parseIpRouteJSON(output string) ([]Route, error) {
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
func (n *netr) convertJSONRouteToRoute(jsonRoute ipRouteJSON) *Route {
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

// parseLinuxRoutesFromProc parses IPv4 routes from /proc/net/route output
// Format: Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
// based on osquery implementation https://github.com/osquery/osquery/blob/master/osquery/tables/networking/linux/routes.cpp
func (n *netr) parseLinuxRoutesFromProc(output string) ([]Route, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, errors.New("invalid /proc/net/route format")
	}

	// Skip header line
	var routes []Route
	for i := 1; i < len(lines); i++ {
		line := strings.Fields(lines[i])
		if len(line) < 11 {
			continue
		}

		iface := line[0]
		destHex := line[1]
		gatewayHex := line[2]
		flagsHex := line[3]
		maskHex := line[7]

		// Parse hex values
		dest, err := hexToIP(destHex)
		if err != nil {
			continue
		}
		gateway, err := hexToIP(gatewayHex)
		if err != nil {
			continue
		}
		mask, err := hexToIP(maskHex)
		if err != nil {
			continue
		}

		destStr := dest.String()
		// Calculate CIDR from netmask
		if mask.To4() != nil {
			ones, _ := net.IPMask(mask.To4()).Size()
			destStr = fmt.Sprintf("%s/%d", dest.String(), ones)
		}

		// Handle default route
		if dest.Equal(net.IPv4zero) {
			destStr = "0.0.0.0/0"
		}

		// Parse flags (hex to int)
		flagsInt, err := strconv.ParseUint(flagsHex, 16, 32)
		if err != nil {
			flagsInt = 0
		}

		// Convert flags to strings using BSD-style route flags (RTF_*)
		flags := parseRouteFlags(int64(flagsInt))

		// Handle gateway
		gatewayStr := gateway.String()
		if gateway.Equal(net.IPv4zero) {
			gatewayStr = ""
		}

		routes = append(routes, Route{
			Destination: destStr,
			Gateway:     gatewayStr,
			Flags:       flags,
			Interface:   iface,
		})
	}

	return routes, nil
}

// parseLinuxIPv6RoutesFromProc parses IPv6 routes from /proc/net/ipv6_route output
// Format: destination dest_prefix_len source src_prefix_len next_hop metric ref use flags device
// Based on osquery implementation https://github.com/osquery/osquery/blob/master/osquery/tables/networking/linux/routes.cpp
func (n *netr) parseLinuxIPv6RoutesFromProc(output string) ([]Route, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return []Route{}, nil
	}

	var routes []Route
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		destHex := fields[0]
		prefixLenHex := fields[1] // Destination prefix length (stored as hex string representing decimal)
		nextHopHex := fields[4]   // Next hop is at index 4
		device := fields[9]

		// Parse destination IPv6 address (32 hex chars = 128 bits)
		dest, err := hexToIPv6(destHex)
		if err != nil {
			continue
		}

		// Parse prefix length (hex string representing decimal value)
		// "00" = 0, "80" = 128 (hex 0x80 = decimal 128)
		prefixLen, err := strconv.ParseInt(prefixLenHex, 16, 32)
		if err != nil {
			continue
		}

		// Format destination with CIDR
		destStr := fmt.Sprintf("%s/%d", dest.String(), int(prefixLen))

		// Handle default route
		if dest.Equal(net.IPv6zero) {
			destStr = "::/0"
		}

		// Parse next hop
		nextHop, err := hexToIPv6(nextHopHex)
		var gatewayStr string
		if err == nil && !nextHop.Equal(net.IPv6zero) {
			gatewayStr = nextHop.String()
		} else {
			gatewayStr = "::"
		}

		// Filter out IPv6 multicast routes (ff00::/8) to match osquery results
		if strings.HasPrefix(destStr, "ff00::") || strings.HasPrefix(destStr, "ff") {
			continue
		}

		routes = append(routes, Route{
			Destination: destStr,
			Gateway:     gatewayStr,
			Flags:       []string{}, // /proc/net/ipv6_route doesn't provide flags in a simple format
			Interface:   device,
		})
	}

	return routes, nil
}

// hexToIP converts a hex string (little-endian) to net.IP (IPv4)
func hexToIP(hexStr string) (net.IP, error) {
	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return nil, err
	}
	// Convert from little-endian to IP
	return net.IPv4(
		byte(val&0xff),
		byte((val>>8)&0xff),
		byte((val>>16)&0xff),
		byte((val>>24)&0xff),
	), nil
}

// hexToIPv6 converts a hex string (32 hex chars) to net.IP (IPv6)
func hexToIPv6(hexStr string) (net.IP, error) {
	if len(hexStr) != 32 {
		return nil, fmt.Errorf("invalid IPv6 hex string length: %d", len(hexStr))
	}

	ip := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		// IPv6 in /proc/net/ipv6_route is stored in network byte order (big-endian)
		// Each byte is represented by 2 hex chars
		byteStr := hexStr[i*2 : i*2+2]
		val, err := strconv.ParseUint(byteStr, 16, 8)
		if err != nil {
			return nil, err
		}
		ip[i] = byte(val)
	}

	return ip, nil
}
