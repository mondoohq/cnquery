// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/iru/provider"
)

var Config = plugin.Provider{
	Name:            "iru",
	ID:              "go.mondoo.com/mql/providers/iru",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "iru",
			Use:   "iru",
			Short: "an Iru tenant",
			Long: `Use the Iru provider to query an Iru (formerly Kandji) tenant.

To access the Iru API, you need your tenant API URL and a bearer token
issued in the Iru admin console. The token's per-endpoint permission flags
determine which sub-resources will return data.

Examples:
  mql shell iru --api-url https://<subdomain>.api.kandji.io --token <api-token>
  cnspec scan iru --api-url https://<subdomain>.api.kandji.io --token <api-token>
`,
			Flags: []plugin.Flag{
				{
					Long:    "api-url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Iru tenant API URL (e.g. https://<subdomain>.api.kandji.io)",
				},
				{
					Long:        "token",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Iru API bearer token",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=iru"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"api": nil,
			},
		},
	},
}
