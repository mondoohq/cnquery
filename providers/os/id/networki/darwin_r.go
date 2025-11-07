// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package networki

import (
	"fmt"
	"net"
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

		// Filter out IPv6 multicast routes (ff00::/8) to match osquery behavior
		if destIP := net.ParseIP(dest); destIP != nil {
			if destIP.To4() == nil && destIP.To16() != nil {
				// IPv6 multicast addresses start with 0xff
				if destIP[0] == 0xff {
					continue
				}
			}
		} else if strings.HasPrefix(dest, "ff") {
			// IPv6 multicast CIDR notation
			continue
		}

		// Filter out IPv6 prefix routes (fe80::%X) that osquery doesn't show
		// osquery only shows specific link-local addresses, not prefix routes
		if strings.HasPrefix(dest, "fe80::%") && !strings.Contains(dest[7:], ":") {
			// This is a prefix route like "fe80::%1", "fe80::%11", etc.
			// osquery doesn't show these, only specific addresses like "fe80::1%lo0"
			continue
		}

		// Filter out IPv6 link-local routes where gateway is another link-local address
		// osquery only shows link-local routes with link#X gateways, not IP gateways
		if strings.HasPrefix(dest, "fe80::") && strings.HasPrefix(gateway, "fe80::") {
			// This is a link-local route with an IP gateway (not link#X)
			// osquery doesn't show these, only routes with link#X gateways
			continue
		}

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       int64(routeMsg.Flags),
			Interface:   iface,
		})
	}

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
	// Index 0: Destination (Inet4Addr/Inet6Addr)
	// Index 1: Gateway (Inet4Addr/Inet6Addr or LinkAddr)
	// Index 2: Netmask (Inet4Addr/Inet6Addr, if present)
	// Index 3: Interface (LinkAddr, if present)

	var destAddr, gatewayAddr, netmaskAddr route.Addr
	var gatewayLinkAddr *route.LinkAddr
	var linkAddr *route.LinkAddr

	// Process addresses in order, respecting their position
	// RouteMessage.Addrs preserves RTA_* order: 0=destination, 1=gateway, 2=netmask, 3=interface
	for i, addr := range routeMsg.Addrs {
		if addr == nil {
			continue
		}

		switch a := addr.(type) {
		case *route.Inet4Addr:
			// Position-based identification: index 0 is destination, index 2+ is netmask
			if i == 0 {
				// First address is always destination (even if all zeros for default route)
				destAddr = addr
			} else if i == 1 && gatewayAddr == nil && gatewayLinkAddr == nil {
				// Second address is gateway
				gatewayAddr = addr
			} else if i >= 2 && n.isValidNetmask(a.IP[:]) && netmaskAddr == nil {
				// Later positions with valid netmask pattern are netmasks
				netmaskAddr = addr
			}
		case *route.Inet6Addr:
			// Position-based identification: index 0 is destination, index 2+ is netmask
			if i == 0 {
				// First address is always destination (even if all zeros for default route)
				destAddr = addr
			} else if i == 1 && gatewayAddr == nil && gatewayLinkAddr == nil {
				// Second address is gateway
				gatewayAddr = addr
			} else if i >= 2 && n.isValidNetmask6(a.IP[:]) && netmaskAddr == nil {
				// Later positions with valid netmask pattern are netmasks
				netmaskAddr = addr
			}
		case *route.LinkAddr:
			// Link address position determines its role
			if i == 1 && gatewayAddr == nil && gatewayLinkAddr == nil {
				// Second position link address is the gateway
				gatewayLinkAddr = a
			} else {
				// Later position is the interface
				if linkAddr == nil {
					linkAddr = a
				}
			}
		}
	}

	// Convert destination address
	if destAddr != nil {
		dest = n.addrToString(destAddr, interfaceMap)
	}

	// Convert gateway address
	if gatewayAddr != nil {
		gateway = n.addrToString(gatewayAddr, interfaceMap)
	} else if gatewayLinkAddr != nil {
		// Gateway is a link address (e.g., "link#11")
		gateway = fmt.Sprintf("link#%d", gatewayLinkAddr.Index)
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
	// Always prefer routeMsg.Index as it represents the actual interface for the route
	if routeMsg.Index > 0 {
		if name, ok := interfaceMap[routeMsg.Index]; ok {
			iface = name
		} else {
			// Fallback to index if name not found
			iface = fmt.Sprintf("%d", routeMsg.Index)
		}
	} else if linkAddr != nil && linkAddr.Name != "" {
		// Fallback to link address name if index not available
		iface = linkAddr.Name
	} else if gatewayLinkAddr != nil && gatewayLinkAddr.Name != "" {
		// Last resort: use gateway link address name
		iface = gatewayLinkAddr.Name
	}

	// Don't normalize default routes - osquery shows them as "0.0.0.0" and "::"
	// Keep them as-is to match osquery output

	return dest, gateway, iface, nil
}

// addrToString converts a route.Addr to a string IP address
// For IPv6 addresses with zone IDs, converts the zone ID to interface name to match osquery format
func (n *neti) addrToString(addr route.Addr, interfaceMap map[int]string) string {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		return net.IPv4(a.IP[0], a.IP[1], a.IP[2], a.IP[3]).String()
	case *route.Inet6Addr:
		ip := make(net.IP, 16)
		copy(ip, a.IP[:])
		if a.ZoneID > 0 {
			// Convert zone ID to interface name to match osquery format (e.g., %16 -> %utun1)
			if ifaceName, ok := interfaceMap[a.ZoneID]; ok {
				return fmt.Sprintf("%s%%%s", ip.String(), ifaceName)
			}
			// Fallback to zone ID number if interface name not found
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
