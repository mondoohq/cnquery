// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/id/machineid"
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

	interfaces, err := networki.Interfaces(conn, platform)
	if err != nil {
		log.Error().Err(err).Msg("unable to get interfaces for routes")
		interfaces = []networki.Interface{}
	}

	// Map interfaces by name
	interfaceMap := make(map[string]networki.Interface)
	for _, iface := range interfaces {
		interfaceMap[iface.Name] = iface
	}

	routeResources := []any{}
	for _, route := range routes {
		var ifaceResource plugin.Resource
		if iface, ok := interfaceMap[route.Interface]; ok {
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
				return nil, err
			}
		}

		// Convert flags to string array
		flagStrings := route.FlagsToStrings()

		routeRes, err := NewResource(c.MqlRuntime, "networkRoute", map[string]*llx.RawData{
			"__id":        llx.StringData(route.Destination + "/" + route.Gateway + "/" + route.Interface),
			"destination": llx.StringData(route.Destination),
			"gateway":     llx.StringData(route.Gateway),
			"flags":       llx.ArrayData(convert.SliceAnyToInterface(flagStrings), types.String),
			"iface":       llx.ResourceData(ifaceResource, "networkInterface"),
		})
		if err != nil {
			log.Debug().Err(err).Msg("unable to create networkRoute resource")
			return nil, err
		}
		routeResources = append(routeResources, routeRes)
	}

	machineID, err := machineid.MachineId(conn, platform)
	if err != nil || machineID == "" {
		// Fallback to asset MRN if machine ID is not available
		assetID := conn.Asset().GetMrn()
		if assetID == "" {
			assetID = "default"
		}
		machineID = assetID
	}

	routesRes, err := NewResource(c.MqlRuntime, "networkRoutes", map[string]*llx.RawData{
		"__id": llx.StringData(fmt.Sprintf("networkRoutes(%s)", machineID)),
		"list": llx.ArrayData(routeResources, types.Resource("networkRoute")),
	})
	if err != nil {
		return nil, err
	}

	return routesRes.(*mqlNetworkRoutes), nil
}
