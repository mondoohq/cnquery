// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/networki"
	"go.mondoo.com/cnquery/v11/types"
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

			ipaddresses := []interface{}{}
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
