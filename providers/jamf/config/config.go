// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/jamf/provider"
)

var Config = plugin.Provider{
	Name:            "jamf",
	ID:              "go.mondoo.com/mql/providers/jamf",
	Version:         "13.1.4",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "jamf",
			Use:   "jamf",
			Short: "a Jamf Pro account",
			Long: `Use the Jamf provider to query a Jamf Pro instance.

To access the Jamf Pro API, you need your instance domain and API credentials.

Examples:
  mql shell jamf --client-id <your-client-id> --client-secret <your-client-secret> --instance-domain https://yourdomain.jamfcloud.com
  cnspec scan jamf --client-id <your-client-id> --client-secret <your-client-secret> --instance-domain https://yourdomain.jamfcloud.com
`,
			Flags: []plugin.Flag{
				{
					Long:    "client-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Jamf Pro API client ID",
				},
				{
					Long:        "client-secret",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Jamf Pro API client secret",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:    "instance-domain",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Jamf Pro domain (e.g., https://yourdomain.jamfcloud.com)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=jamf"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"api": nil,
			},
		},
	},
}
