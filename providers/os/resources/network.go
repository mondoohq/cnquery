// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/id/networki"
	"go.mondoo.com/cnquery/v12/types"
)

func (c *mqlNetwork) interfaces() ([]any, error) {
	log.Debug().Msg("os.network> interfaces")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network interfaces")
		c.Interfaces = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	var resources []any
	if interfaces != nil {
		var resource plugin.Resource
		for _, neti := range interfaces {
			log.Debug().Interface("interface", neti).Msg("os.network> adding interface")

			ipaddresses := []any{}
			for _, ipaddress := range neti.IPAddresses {
				resource, err = NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
					"__id":      llx.StringData(ipaddress.IP.String()),
					"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
					"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
					"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
					"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
					"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
				})
				if err != nil { // non-critical error
					log.Error().Err(err).Msg("unable to create ipaddress resource")
					continue
				}
				ipaddresses = append(ipaddresses, resource)
			}

			resource, err = NewResource(c.MqlRuntime, "networkInterface", map[string]*llx.RawData{
				"__id":    llx.StringData(neti.Name + "/" + neti.MACAddress),
				"name":    llx.StringData(neti.Name),
				"mac":     llx.StringData(neti.MACAddress),
				"mtu":     llx.IntData(neti.MTU),
				"active":  llx.BoolDataPtr(neti.Active),
				"virtual": llx.BoolDataPtr(neti.Virtual),
				"vendor":  llx.StringData(neti.Vendor),
				"flags":   llx.ArrayData(convert.SliceAnyToInterface(neti.Flags), types.String),
				"ips":     llx.ArrayData(ipaddresses, types.Resource("ipAddress")),
			})
			if err != nil {
				return resources, nil
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (c *mqlNetwork) routes() (*mqlNetworkRoutes, error) {
	log.Debug().Msg("os.network> routes")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	routes, err := networki.Routes(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network routes")
		return nil, err
	}

	// Get interfaces to map route interfaces
	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to get interfaces for routes")
		interfaces = []networki.Interface{}
	}

	// Create interface map by name
	interfaceMap := make(map[string]networki.Interface)
	for _, iface := range interfaces {
		interfaceMap[iface.Name] = iface
	}

	// Convert routes to resources
	routeResources := []any{}
	for _, route := range routes {
		// Find the interface for this route
		var ifaceResource plugin.Resource
		if iface, ok := interfaceMap[route.Interface]; ok {
			// Create networkInterface resource
			ipaddresses := []any{}
			for _, ipaddress := range iface.IPAddresses {
				ipRes, err := NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
					"__id":      llx.StringData(ipaddress.IP.String()),
					"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
					"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
					"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
					"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
					"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
				})
				if err != nil {
					continue
				}
				ipaddresses = append(ipaddresses, ipRes)
			}

			ifaceResource, err = NewResource(c.MqlRuntime, "networkInterface", map[string]*llx.RawData{
				"__id":    llx.StringData(iface.Name + "/" + iface.MACAddress),
				"name":    llx.StringData(iface.Name),
				"mac":     llx.StringData(iface.MACAddress),
				"mtu":     llx.IntData(iface.MTU),
				"active":  llx.BoolDataPtr(iface.Active),
				"virtual": llx.BoolDataPtr(iface.Virtual),
				"vendor":  llx.StringData(iface.Vendor),
				"flags":   llx.ArrayData(convert.SliceAnyToInterface(iface.Flags), types.String),
				"ips":     llx.ArrayData(ipaddresses, types.Resource("ipAddress")),
			})
			if err != nil {
				log.Debug().Err(err).Str("iface", route.Interface).Msg("unable to create networkInterface resource")
			}
		}

		routeRes, err := NewResource(c.MqlRuntime, "networkRoute", map[string]*llx.RawData{
			"__id":        llx.StringData(route.Destination + "/" + route.Gateway + "/" + route.Interface),
			"destination": llx.StringData(route.Destination),
			"gateway":     llx.StringData(route.Gateway),
			"flags":       llx.IntData(route.Flags),
			"iface":       llx.ResourceData(ifaceResource, "networkInterface"),
		})
		if err != nil {
			log.Debug().Err(err).Msg("unable to create networkRoute resource")
			continue
		}
		routeResources = append(routeResources, routeRes)
	}

	// Create networkRoutes resource
	routesRes, err := NewResource(c.MqlRuntime, "networkRoutes", map[string]*llx.RawData{
		"__id": llx.StringData("networkRoutes"),
		"list": llx.ArrayData(routeResources, types.Resource("networkRoute")),
	})
	if err != nil {
		return nil, err
	}

	return routesRes.(*mqlNetworkRoutes), nil
}

func (c *mqlNetwork) ipv4() ([]any, error) {
	log.Debug().Msg("os.network> ipv4")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network interfaces")
		return nil, nil
	}

	var ipv4Addresses []any
	for _, neti := range interfaces {
		for _, ipaddress := range neti.IPAddresses {
			if version, ok := ipaddress.Version(); ok && version == networki.IPv4 {
				resource, err := NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
					"__id":      llx.StringData(ipaddress.IP.String()),
					"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
					"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
					"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
					"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
					"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
				})
				if err != nil {
					log.Error().Err(err).Msg("unable to create ipaddress resource")
					continue
				}
				ipv4Addresses = append(ipv4Addresses, resource)
			}
		}
	}

	return ipv4Addresses, nil
}

func (c *mqlNetwork) ipv6() ([]any, error) {
	log.Debug().Msg("os.network> ipv6")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network interfaces")
		return nil, nil
	}

	var ipv6Addresses []any
	for _, neti := range interfaces {
		for _, ipaddress := range neti.IPAddresses {
			if version, ok := ipaddress.Version(); ok && version == networki.IPv6 {
				resource, err := NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
					"__id":      llx.StringData(ipaddress.IP.String()),
					"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
					"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
					"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
					"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
					"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
				})
				if err != nil {
					log.Error().Err(err).Msg("unable to create ipaddress resource")
					continue
				}
				ipv6Addresses = append(ipv6Addresses, resource)
			}
		}
	}

	return ipv6Addresses, nil
}

func (c *mqlNetwork) primaryIpv4() (*mqlIpAddress, error) {
	log.Debug().Msg("os.network> primaryIpv4")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	// Get routes to find default route
	routes, err := networki.Routes(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network routes")
		return nil, err
	}

	// Find default IPv4 route
	var defaultRoute *networki.Route
	for i := range routes {
		if routes[i].IsDefaultRoute() && routes[i].IsIPv4() {
			defaultRoute = &routes[i]
			break
		}
	}

	if defaultRoute == nil {
		log.Debug().Msg("os.network> no default IPv4 route found")
		return nil, nil
	}

	// Get interfaces to find the IP on the default route interface
	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network interfaces")
		return nil, err
	}

	// Find the interface for the default route
	for _, iface := range interfaces {
		if iface.Name == defaultRoute.Interface {
			// Find first IPv4 address on this interface
			for _, ipaddress := range iface.IPAddresses {
				if version, ok := ipaddress.Version(); ok && version == networki.IPv4 {
					resource, err := NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
						"__id":      llx.StringData(ipaddress.IP.String()),
						"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
						"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
						"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
						"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
						"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
					})
					if err != nil {
						return nil, err
					}
					return resource.(*mqlIpAddress), nil
				}
			}
		}
	}

	return nil, nil
}

func (c *mqlNetwork) primaryIpv6() (*mqlIpAddress, error) {
	log.Debug().Msg("os.network> primaryIpv6")
	conn := c.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	// Get routes to find default route
	routes, err := networki.Routes(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network routes")
		return nil, err
	}

	// Find default IPv6 route
	var defaultRoute *networki.Route
	for i := range routes {
		if routes[i].IsDefaultRoute() && routes[i].IsIPv6() {
			defaultRoute = &routes[i]
			break
		}
	}

	if defaultRoute == nil {
		log.Debug().Msg("os.network> no default IPv6 route found")
		return nil, nil
	}

	// Get interfaces to find the IP on the default route interface
	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to detect network interfaces")
		return nil, err
	}

	// Find the interface for the default route
	for _, iface := range interfaces {
		if iface.Name == defaultRoute.Interface {
			// Find first IPv6 address on this interface
			for _, ipaddress := range iface.IPAddresses {
				if version, ok := ipaddress.Version(); ok && version == networki.IPv6 {
					resource, err := NewResource(c.MqlRuntime, "ipAddress", map[string]*llx.RawData{
						"__id":      llx.StringData(ipaddress.IP.String()),
						"ip":        llx.IPData(llx.ParseIP(ipaddress.IP.String())),
						"cidr":      llx.IPData(llx.ParseIP(ipaddress.CIDR)),
						"broadcast": llx.IPData(llx.ParseIP(ipaddress.Broadcast)),
						"gateway":   llx.IPData(llx.ParseIP(ipaddress.Gateway)),
						"subnet":    llx.IPData(llx.ParseIP(ipaddress.Subnet)),
					})
					if err != nil {
						return nil, err
					}
					return resource.(*mqlIpAddress), nil
				}
			}
		}
	}

	return nil, nil
}

func (c *mqlNetworkInterface) externalIP() (llx.RawIP, error) {
	log.Debug().Str("interface", c.Name.Data).Msg("os.network.interface> externalIP")

	// TODO: Implement external IP service lookup
	// whatismyip.mondoo.com (when implemented)

	return llx.RawIP{}, nil
}
