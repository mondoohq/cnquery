// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/provider"
)

var Config = plugin.Provider{
	Name:            "cloudflare",
	ID:              "go.mondoo.com/cnquery/v11/providers/cloudflare",
	Version:         "11.0.25",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "cloudflare",
			Use:   "cloudflare",
			Short: "Cloudflare provider",
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryAuto,
				connection.DiscoveryZones,
				connection.DiscoveryAccounts,
			},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Cloudflare access token",
				},
			},
		},
	},
}
