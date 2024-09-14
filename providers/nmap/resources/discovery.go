// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/nmap/connection"
	"go.mondoo.com/cnquery/v11/utils/stringx"
)

func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.NmapConnection)
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

	for i := range networks {
		network := networks[i]

		targetResource, err := runtime.CreateResource(runtime, "nmap.network ", map[string]*llx.RawData{
			"target": llx.StringData(network),
		})
		if err != nil {
			return nil, err
		}
		hosts := targetResource.(*mqlNmapNetwork).GetHosts().Data
		for i := range hosts {
			entry := hosts[i]
			host := entry.(*mqlNmapHost)

			a := &inventory.Asset{
				Name: host.GetName().Data,
				Connections: []*inventory.Config{
					{
						Type:        "nmap",
						Host:        host.GetName().Data,
						Credentials: conf.Credentials,
					},
				},
			}

			assetList = append(assetList, a)
		}
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
