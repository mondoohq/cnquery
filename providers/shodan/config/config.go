// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
	"go.mondoo.com/cnquery/v11/providers/shodan/provider"
)

var Config = plugin.Provider{
	Name:            "shodan",
	ID:              "go.mondoo.com/cnquery/v11/providers/shodan",
	Version:         "11.0.76",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "shodan",
			Use:   "shodan",
			Short: "a Shodan account",
			Long: `Use the shodan provider to query domain and IP security information in the Shodan search engine. 

If you set the SHODAN_TOKEN environment variable, you can omit the token flag.

Examples:
  cnquery shell shodan --token <api-token>
	cnquery shell shodan --networks <ip-range> --discover hosts
`,
			MinArgs: 0,
			MaxArgs: 2,
			Discovery: []string{
				connection.DiscoveryHosts,
			},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Shodan API token",
				},
				{
					Long:    "networks",
					Type:    plugin.FlagType_List,
					Default: "",
					Desc:    "Only include matching networks",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=shodan"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"host":   nil,
				"domain": nil,
				"org":    nil,
			},
		},
	},
}
