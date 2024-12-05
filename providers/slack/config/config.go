// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/slack/provider"
)

var Config = plugin.Provider{
	Name:            "slack",
	ID:              "go.mondoo.com/cnquery/v9/providers/slack",
	Version:         "11.0.47",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "slack",
			Use:   "slack",
			Short: "a Slack team",
			Long: `Use the slack provider to query resources in a Slack team.

Examples:
  cnquery shell slack --token <api-token>
  cnspec scan slack --team-id <team-id> --token <api-token>
`,
			MinArgs:   0,
			MaxArgs:   0,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Slack API token (To learn more, read https://mondoo.com/docs/cnspec/saas/slack/)",
				},
				{
					Long:    "team-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Team ID (required if the token is an org token)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=slack"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"team": nil,
			},
		},
	},
}
