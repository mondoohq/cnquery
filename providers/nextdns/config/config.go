// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/nextdns/connection"
	"go.mondoo.com/mql/v13/providers/nextdns/provider"
)

var Config = plugin.Provider{
	Name:            "nextdns",
	ID:              "go.mondoo.com/mql/providers/nextdns",
	Version:         "13.0.6",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "nextdns",
			Use:   "nextdns",
			Short: "a NextDNS account",
			Long: `Use the nextdns provider to query resources in your NextDNS account, including profiles and their security, privacy, and parental-control configuration.

Examples:
  cnspec shell nextdns --api-key <api_key>
  cnspec scan nextdns --api-key <api_key>

Notes:
  If you set the NEXTDNS_API_KEY environment variable, you can omit the api-key flag.
`,
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryAuto,
				connection.DiscoveryAccounts,
				connection.DiscoveryProfiles,
			},
			Flags: []plugin.Flag{
				{
					Long:    "api-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "NextDNS API key for authentication",
				},
			},
		},
	},
}
