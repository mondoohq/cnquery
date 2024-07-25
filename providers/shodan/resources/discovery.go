// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/netip"
	"strings"

	"github.com/shadowscatcher/shodan/search"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
	"go.mondoo.com/cnquery/v11/utils/stringx"
)

func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.ShodanConnection)
	if conn == nil || conn.Asset() == nil || len(conn.Asset().Connections) == 0 {
		return nil, nil
	}

	conf := conn.Asset().Connections[0]
	targets := handleTargets(conf.Discover.Targets)
	if !stringx.ContainsAnyOf(targets, connection.DiscoveryHosts, connection.DiscoveryAll, connection.DiscoveryAuto) {
		return nil, nil
	}

	// we only need to discover when networks are specified
	networkValue, ok := conf.Options["networks"]
	if !ok || networkValue == "" {
		return nil, nil
	}
	networks := strings.Split(networkValue, ",")
	assetList := []*inventory.Asset{}

	addrs := resolveNetworks(networks)

	for i := range addrs {
		addr := addrs[i]

		// if the host is not found, we skip it
		// TODO: optimize this by using the bulk search so we save some API calls
		_, err := conn.Client().Host(context.Background(), search.HostParams{
			IP: addr.String(),
		})
		if err != nil {
			continue
		}

		a := &inventory.Asset{
			Name: addr.String(),
			Connections: []*inventory.Config{
				{
					Type: "shodan",
					Host: addr.String(),
					Options: map[string]string{
						"search": "host",
					},
					Credentials: conf.Credentials,
				},
			},
		}

		assetList = append(assetList, a)
	}

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: assetList,
	}}
	return in, nil
}

func handleTargets(targets []string) []string {
	if stringx.Contains(targets, connection.DiscoveryAll) {
		return []string{
			connection.DiscoveryHosts,
		}
	}
	return targets
}

// resolve all IPs in the CIDR range
func resolveNetworks(networks []string) []netip.Addr {
	addresses := []netip.Addr{}
	for i := range networks {
		network := networks[i]
		// check if network is a CIDR range
		if strings.Contains(network, "/") {
			ips, err := cidrIPs(network)
			if err == nil {
				addresses = append(addresses, ips...)
			}
		} else {
			// we assume a single IP address
			addr, err := netip.ParseAddr(network)
			if err == nil {
				addresses = append(addresses, addr)
			}
		}
	}
	return addresses
}

// cidrIPs determines the ips from a CIDR range
func cidrIPs(cidr string) ([]netip.Addr, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}

	var ips []netip.Addr
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		ips = append(ips, addr)
	}

	if len(ips) < 2 {
		return ips, nil
	}

	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}
