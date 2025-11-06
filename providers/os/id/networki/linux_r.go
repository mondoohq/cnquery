// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// detectLinuxRoutes detects network routes on Linux by reading /proc/net/route
func (n *neti) detectLinuxRoutes() ([]Route, error) {
	fs := n.connection.FileSystem()
	file, err := fs.Open("/proc/net/route")
	if err != nil {
		// Fallback to command if file is not accessible (e.g., SSH without SFTP)
		log.Debug().Err(err).Msg("os.network.routes> could not open /proc/net/route, trying command")
		return n.detectLinuxRoutesViaCommand()
	}
	defer file.Close()

	return n.parseProcNetRoute(file)
}

// detectLinuxRoutesViaCommand uses 'ip route show' as fallback
func (n *neti) detectLinuxRoutesViaCommand() ([]Route, error) {
	output, err := n.RunCommand("ip route show")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via ip command")
	}

	routes, err := n.parseIpRouteOutput(output)
	if err != nil {
		return nil, err
	}

	// Also get IPv6 routes
	ipv6Output, err := n.RunCommand("ip -6 route show")
	if err == nil {
		ipv6Routes, err := n.parseIpRouteOutput(ipv6Output)
		if err == nil {
			routes = append(routes, ipv6Routes...)
		}
	}

	return routes, nil
}

// parseProcNetRoute parses /proc/net/route file format
// Format: Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
func (n *neti) parseProcNetRoute(r io.Reader) ([]Route, error) {
	var routes []Route
	scanner := bufio.NewScanner(r)

	// Skip header line
	if !scanner.Scan() {
		return routes, nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		iface := fields[0]
		destHex := fields[1]
		gatewayHex := fields[2]
		flagsHex := fields[3]

		// Parse destination (hex to IP)
		destIP, err := hexToIP(destHex)
		if err != nil {
			log.Debug().Err(err).Str("dest", destHex).Msg("os.network.routes> could not parse destination")
			continue
		}

		// Parse gateway (hex to IP)
		gatewayIP, err := hexToIP(gatewayHex)
		if err != nil {
			log.Debug().Err(err).Str("gateway", gatewayHex).Msg("os.network.routes> could not parse gateway")
			continue
		}

		// Parse flags (hex to int)
		flags, err := strconv.ParseInt(flagsHex, 16, 64)
		if err != nil {
			log.Debug().Err(err).Str("flags", flagsHex).Msg("os.network.routes> could not parse flags")
			continue
		}

		// Parse mask if available
		var dest string
		if len(fields) >= 7 {
			maskHex := fields[7]
			maskIP, err := hexToIP(maskHex)
			if err == nil {
				mask := net.IPMask(maskIP.To4())
				ones, _ := mask.Size()
				if destIP.Equal(net.IPv4zero) {
					dest = "0.0.0.0/0"
				} else {
					dest = fmt.Sprintf("%s/%d", destIP.String(), ones)
				}
			} else {
				dest = destIP.String()
			}
		} else {
			if destIP.Equal(net.IPv4zero) {
				dest = "0.0.0.0/0"
			} else {
				dest = destIP.String()
			}
		}

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gatewayIP.String(),
			Flags:       flags,
			Interface:   iface,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error reading /proc/net/route")
	}

	// Also get IPv6 routes via command (IPv6 routes are not in /proc/net/route)
	ipv6Routes, err := n.detectLinuxIPv6Routes()
	if err == nil {
		routes = append(routes, ipv6Routes...)
	}

	return routes, nil
}

// detectLinuxIPv6Routes gets IPv6 routes via command
func (n *neti) detectLinuxIPv6Routes() ([]Route, error) {
	output, err := n.RunCommand("ip -6 route show")
	if err != nil {
		return nil, nil // IPv6 routes are optional
	}

	return n.parseIpRouteOutput(output)
}

// parseIpRouteOutput parses output from 'ip route show' command
func (n *neti) parseIpRouteOutput(output string) ([]Route, error) {
	var routes []Route
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		route := Route{}

		// Parse destination
		if fields[0] == "default" || isDefaultRoute(fields[0]) {
			// Check if it's IPv6 by looking at the gateway or interface
			if len(fields) > 2 && fields[1] == "via" && strings.Contains(fields[2], ":") {
				route.Destination = "::/0"
			} else {
				route.Destination = "0.0.0.0/0"
			}
		} else {
			route.Destination = fields[0]
		}

		// Parse gateway (via <ip>)
		if len(fields) > 2 && fields[1] == "via" {
			route.Gateway = fields[2]
		}

		// Parse interface (dev <name>)
		for i, field := range fields {
			if field == "dev" && i+1 < len(fields) {
				route.Interface = fields[i+1]
				break
			}
		}

		if route.Interface == "" {
			continue
		}

		routes = append(routes, route)
	}

	return routes, scanner.Err()
}

// hexToIP converts a hexadecimal string (little-endian) to an IP address
func hexToIP(hexStr string) (net.IP, error) {
	// Remove leading zeros if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimPrefix(hexStr, "0X")

	// Parse hex string
	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse hex string")
	}

	// Convert to bytes (little-endian)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(val))

	// Return as IPv4
	return net.IP(bytes), nil
}
