// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/grafana/provider"
)

var Config = plugin.Provider{
	Name:            "grafana",
	ID:              "go.mondoo.com/mql/v13/providers/grafana",
	Version:         "13.1.8",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "grafana",
			Use:   "grafana",
			Short: "a Grafana organization",
			Long: `Use the grafana provider to query resources in a Grafana organization.

Examples:
  cnspec shell grafana --url https://myorg.grafana.net --token <api-token>
  cnspec scan grafana --url https://myorg.grafana.net --token <api-token>

Notes:
  If you set the GRAFANA_TOKEN environment variable, you can omit the token flag.
  If you set the GRAFANA_URL environment variable, you can omit the url flag.
`,
			MinArgs:   0,
			MaxArgs:   0,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Grafana service account token",
				},
				{
					Long:    "url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Grafana instance URL (e.g., https://myorg.grafana.net)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=grafana"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"org": nil,
			},
		},
	},
}
