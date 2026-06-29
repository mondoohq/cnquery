// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
	"go.mondoo.com/mql/v13/providers/digitalocean/provider"
)

var Config = plugin.Provider{
	Name:            "digitalocean",
	ID:              "go.mondoo.com/mql/providers/digitalocean",
	Version:         "13.10.1",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "digitalocean",
			Use:   "digitalocean",
			Short: "a DigitalOcean account",
			Long: `Use the digitalocean provider to query resources in a DigitalOcean account.

Examples:
  mql shell digitalocean --token <api-token>
  mql shell digitalocean

Notes:
  If you set the DIGITALOCEAN_TOKEN environment variable, you can omit the token flag.
`,
			MinArgs: 0,
			MaxArgs: 0,
			Discovery: []string{
				connection.DiscoveryAuto,
				connection.DiscoveryAll,
				connection.DiscoveryDatabases,
				connection.DiscoveryKubernetes,
				connection.DiscoveryLoadBalancers,
				connection.DiscoveryFirewalls,
				connection.DiscoverySpacesBuckets,
			},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "DigitalOcean personal access token (env: DIGITALOCEAN_TOKEN)",
				},
			},
		},
	},
}
