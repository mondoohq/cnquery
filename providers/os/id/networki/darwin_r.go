// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

// detectDarwinRoutes detects network routes on macOS using golang.org/x/net/route
func (n *neti) detectDarwinRoutes() ([]Route, error) {
	// Get IPv4 routes
	ipv4Routes, err := n.fetchDarwinRoutes(unix.AF_INET)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get IPv4 routes")
	}

	// Get IPv6 routes
	ipv6Routes, err := n.fetchDarwinRoutes(unix.AF_INET6)
	if err != nil {
		// IPv6 routes are optional, log but don't fail
		log.Debug().Err(err).Msg("failed to get IPv6 routes")
	}
	routes := append(ipv4Routes, ipv6Routes...)

	return routes, nil
}

// fetchDarwinRoutes fetches routes for a specific address family
func (n *neti) fetchDarwinRoutes(af int) ([]Route, error) {
	// Fetch routing information base
	rib, err := route.FetchRIB(af, route.RIBTypeRoute, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch RIB")
	}

	// Parse the RIB
	messages, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse RIB")
	}

	// Get interface map to resolve interface indices to names
	interfaceMap, err := n.getDarwinInterfaceMap()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get interface map, using indices")
		interfaceMap = make(map[int]string)
	}

	var routes []Route
	for _, msg := range messages {
		routeMsg, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		// Extract route information
		dest, gateway, iface, err := n.parseRouteMessage(routeMsg, interfaceMap)
		if err != nil {
			log.Debug().Err(err).Msg("failed to parse route message")
			continue
		}

		if dest == "" {
			continue
		}

		// Filter out unwanted routes
		// TODO: maybe we don't need this, check on osquery what is included
		if n.shouldSkipRoute(dest, gateway) {
			continue
		}

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       int64(routeMsg.Flags),
			Interface:   iface,
		})
	}

	routes = n.deduplicateRoutes(routes)

	return routes, nil
}

// getDarwinInterfaceMap creates a map of interface index to interface name
func (n *neti) getDarwinInterfaceMap() (map[int]string, error) {
	interfaceMap := make(map[int]string)

	// Fetch interface list
	rib, err := route.FetchRIB(unix.AF_UNSPEC, route.RIBTypeInterface, 0)
	if err != nil {
		return nil, err
	}

	messages, err := route.ParseRIB(route.RIBTypeInterface, rib)
	if err != nil {
		return nil, err
	}

	for _, msg := range messages {
		if ifMsg, ok := msg.(*route.InterfaceMessage); ok {
			interfaceMap[ifMsg.Index] = ifMsg.Name
		}
	}

	return interfaceMap, nil
}

// parseRouteMessage extracts destination, gateway, and interface from a RouteMessage
func (n *neti) parseRouteMessage(routeMsg *route.RouteMessage, interfaceMap map[int]string) (dest, gateway, iface string, err error) {
	// RouteMessage.Addrs contains addresses in specific order on macOS:
	// Typically: destination, gateway, netmask, interface, broadcast
	// But we check types to be safe

	var destAddr, gatewayAddr, netmaskAddr route.Addr
	var linkAddr *route.LinkAddr

	for _, addr := range routeMsg.Addrs {
		if addr == nil {
			continue
		}

		switch a := addr.(type) {
		case *route.Inet4Addr:
			// First IPv4 address is typically destination
			if destAddr == nil {
				destAddr = addr
			} else if gatewayAddr == nil {
				// Second IPv4 address is typically gateway
				gatewayAddr = addr
			} else if netmaskAddr == nil {
				// Third IPv4 address is typically netmask
				netmaskAddr = addr
			}
		case *route.Inet6Addr:
			// First IPv6 address is typically destination
			if destAddr == nil {
				destAddr = addr
			} else if gatewayAddr == nil {
				// Second IPv6 address is typically gateway
				gatewayAddr = addr
			}
		case *route.LinkAddr:
			// Link address represents the interface
			linkAddr = a
		}
	}

	// Convert destination address
	if destAddr != nil {
		dest = n.addrToString(destAddr)
	}

	// Convert gateway address
	if gatewayAddr != nil {
		gateway = n.addrToString(gatewayAddr)
	}

	// Calculate CIDR from netmask if available
	// Only use netmask if it's a valid subnet mask
	if dest != "" && netmaskAddr != nil {
		if inet4Addr, ok := netmaskAddr.(*route.Inet4Addr); ok {
			if n.isValidNetmask(inet4Addr.IP[:]) {
				ones := n.netmaskToCIDR(inet4Addr.IP)
				if ones > 0 && ones <= 32 {
					if destIP := net.ParseIP(dest); destIP != nil && destIP.To4() != nil {
						dest = fmt.Sprintf("%s/%d", dest, ones)
					}
				}
			}
		} else if inet6Addr, ok := netmaskAddr.(*route.Inet6Addr); ok {
			if n.isValidNetmask6(inet6Addr.IP[:]) {
				ones := n.netmaskToCIDR6(inet6Addr.IP)
				if ones > 0 && ones <= 128 {
					if destIP := net.ParseIP(dest); destIP != nil && destIP.To4() == nil {
						dest = fmt.Sprintf("%s/%d", dest, ones)
					}
				}
			}
		}
	}

	// Get interface name
	if linkAddr != nil && linkAddr.Name != "" {
		iface = linkAddr.Name
	} else if routeMsg.Index > 0 {
		// Try to get interface name from index
		if name, ok := interfaceMap[routeMsg.Index]; ok {
			iface = name
		} else {
			// Fallback to index if name not found
			iface = fmt.Sprintf("%d", routeMsg.Index)
		}
	}

	// Normalize default route
	if dest == "0.0.0.0" {
		dest = "0.0.0.0/0"
	} else if dest == "::" {
		dest = "::/0"
	}

	return dest, gateway, iface, nil
}

// addrToString converts a route.Addr to a string IP address
func (n *neti) addrToString(addr route.Addr) string {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		return net.IPv4(a.IP[0], a.IP[1], a.IP[2], a.IP[3]).String()
	case *route.Inet6Addr:
		ip := make(net.IP, 16)
		copy(ip, a.IP[:])
		if a.ZoneID > 0 {
			return fmt.Sprintf("%s%%%d", ip.String(), a.ZoneID)
		}
		return ip.String()
	default:
		return ""
	}
}

// netmaskToCIDR converts an IPv4 netmask to CIDR prefix length
func (n *neti) netmaskToCIDR(mask [4]byte) int {
	ones := 0
	for _, b := range mask {
		for i := 0; i < 8; i++ {
			if b&(1<<(7-i)) != 0 {
				ones++
			} else {
				return ones
			}
		}
	}
	return ones
}

// netmaskToCIDR6 converts an IPv6 netmask to CIDR prefix length
func (n *neti) netmaskToCIDR6(mask [16]byte) int {
	ones := 0
	for _, b := range mask {
		if b == 0xff {
			ones += 8
		} else {
			for i := 0; i < 8; i++ {
				if b&(1<<(7-i)) != 0 {
					ones++
				} else {
					return ones
				}
			}
		}
	}
	return ones
}

// isValidNetmask checks if a byte array represents a valid IPv4 netmask
// A valid netmask has all 1s followed by all 0s
func (n *neti) isValidNetmask(mask []byte) bool {
	if len(mask) != 4 {
		return false
	}
	seenZero := false
	for _, b := range mask {
		if seenZero {
			if b != 0 {
				return false
			}
		} else {
			if b == 0xff {
				continue
			} else if b == 0 {
				seenZero = true
			} else {
				// Check if it's a valid partial byte (e.g., 11111100 = 252)
				// All 1s must be on the left, all 0s on the right
				for i := 0; i < 8; i++ {
					if b&(1<<(7-i)) == 0 {
						// Found a 0, all remaining bits must be 0
						if b&((1<<(7-i))-1) != 0 {
							return false
						}
						seenZero = true
						break
					}
				}
			}
		}
	}
	return true
}

// isValidNetmask6 checks if a byte array represents a valid IPv6 netmask
func (n *neti) isValidNetmask6(mask []byte) bool {
	if len(mask) != 16 {
		return false
	}
	seenZero := false
	for _, b := range mask {
		if seenZero {
			if b != 0 {
				return false
			}
		} else {
			if b == 0xff {
				continue
			} else if b == 0 {
				seenZero = true
			} else {
				// Check if it's a valid partial byte
				for i := 0; i < 8; i++ {
					if b&(1<<(7-i)) == 0 {
						if b&((1<<(7-i))-1) != 0 {
							return false
						}
						seenZero = true
						break
					}
				}
			}
		}
	}
	return true
}

// TODO: maybe we don't need this, check on osquery what is included
// shouldSkipRoute determines if a route should be filtered out
func (n *neti) shouldSkipRoute(dest, gateway string) bool {
	// Parse destination to check if it's link-local or multicast
	destIP := net.ParseIP(dest)
	if destIP == nil {
		// Try parsing as CIDR
		ip, _, err := net.ParseCIDR(dest)
		if err != nil {
			return true // Invalid destination, skip
		}
		destIP = ip
	}

	// Skip link-local IPv6 addresses (fe80::/10)
	if destIP.To4() == nil && destIP.To16() != nil {
		if destIP[0] == 0xfe && (destIP[1]&0xc0) == 0x80 {
			return true
		}
		// Skip multicast IPv6 addresses (ff00::/8)
		if destIP[0] == 0xff {
			return true
		}
	}

	// Skip IPv4 multicast addresses (224.0.0.0/4)
	if destIP.To4() != nil {
		if destIP[0] >= 224 && destIP[0] <= 239 {
			return true
		}
		// Skip broadcast addresses
		if destIP.Equal(net.IPv4bcast) {
			return true
		}
		// Skip 169.254.0.0/16 (link-local IPv4)
		if destIP[0] == 169 && destIP[1] == 254 {
			return true
		}
	}

	// Skip host-specific routes (individual IPs) unless they're localhost or have a gateway
	// Default routes are always included
	if dest != "0.0.0.0/0" && dest != "::/0" {
		// Check if it's a host route (no CIDR or /32 or /128)
		if !strings.Contains(dest, "/") {
			// Host route without CIDR - only include if it's localhost or has a gateway
			if !destIP.IsLoopback() && gateway == "" {
				return true
			}
		} else {
			// Check CIDR prefix
			parts := strings.Split(dest, "/")
			if len(parts) == 2 {
				prefix, err := strconv.Atoi(parts[1])
				if err == nil {
					// Skip host routes (/32 for IPv4, /128 for IPv6) unless they're meaningful
					if (destIP.To4() != nil && prefix == 32) || (destIP.To4() == nil && prefix == 128) {
						// Only include host routes if they're localhost or have a gateway
						if !destIP.IsLoopback() && gateway == "" {
							return true
						}
					}
					// Skip routes with invalid prefix lengths (like /2)
					if prefix < 8 || (destIP.To4() != nil && prefix > 32) || (destIP.To4() == nil && prefix > 128) {
						return true
					}
				}
			}
		}
	}

	return false
}

// deduplicateRoutes removes duplicate routes, keeping only the most useful ones
func (n *neti) deduplicateRoutes(routes []Route) []Route {
	// Use a map to track seen routes
	seen := make(map[string]Route)
	var deduplicated []Route

	for _, route := range routes {
		// For default routes, only keep one per address family
		if route.Destination == "0.0.0.0/0" || route.Destination == "::/0" {
			key := route.Destination // Use just destination as key for default routes
			if existing, exists := seen[key]; exists {
				// Prefer route with a gateway, or keep the first one
				if route.Gateway != "" && existing.Gateway == "" {
					seen[key] = route
				}
				// Otherwise keep the existing one
			} else {
				seen[key] = route
			}
		} else {
			// For non-default routes, use destination+gateway+interface as key
			key := fmt.Sprintf("%s|%s|%s", route.Destination, route.Gateway, route.Interface)
			if _, exists := seen[key]; !exists {
				seen[key] = route
				deduplicated = append(deduplicated, route)
			}
		}
	}

	// Add the deduplicated default routes
	for _, route := range seen {
		if route.Destination == "0.0.0.0/0" || route.Destination == "::/0" {
			deduplicated = append(deduplicated, route)
		}
	}

	return deduplicated
}
