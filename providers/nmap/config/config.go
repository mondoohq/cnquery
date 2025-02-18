// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/nmap/connection"
	"go.mondoo.com/cnquery/v11/providers/nmap/provider"
)

var Config = plugin.Provider{
	Name:            "nmap",
	ID:              "go.mondoo.com/cnquery/v11/providers/nmap",
	Version:         "11.0.14",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:    "nmap",
			Use:     "nmap",
			Short:   "a Nmap network scanner",
			Long:    `Use the nmap provider to query network information using the Nmap network scanner.`,
			MinArgs: 0,
			MaxArgs: 2,
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryAuto,
				connection.DiscoveryHosts,
			},
			Flags: []plugin.Flag{
				{
					Long:    "networks",
					Type:    plugin.FlagType_List,
					Default: "",
					Desc:    "Only include repositories with matching names",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=nmap"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"host": nil,
			},
		},
	},
}
