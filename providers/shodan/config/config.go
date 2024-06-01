// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
	"go.mondoo.com/cnquery/v11/providers/shodan/provider"
)

var Config = plugin.Provider{
	Name:            "shodan",
	ID:              "go.mondoo.com/cnquery/v11/providers/shodan",
	Version:         "11.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:    "shodan",
			Use:     "shodan",
			Short:   "a Shodan account",
			MinArgs: 0,
			MaxArgs: 2,
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryAuto,
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
					Desc:    "Only include repositories with matching names.",
				},
			},
		},
	},
}
