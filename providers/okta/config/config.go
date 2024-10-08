// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/okta/provider"
)

var Config = plugin.Provider{
	Name:            "okta",
	ID:              "go.mondoo.com/cnquery/v9/providers/okta",
	Version:         "11.0.33",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "okta",
			Use:   "okta",
			Short: "an Okta organization",
			Long: `Use the okta provider to query resources in an Okta organization.

To query an Okta organization, you need the organization's domain and an API token to access that domain. To learn how, read https://mondoo.com/docs/cnquery/saas/okta/.

Examples:
  cnquery shell okta -organization <okta-domain> -token <api-token>
	cnspec scan okta -organization <okta-domain> -token <api-token>
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "organization",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The domain of the Okta organization to scan",
				},
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Access token for the Okta organization",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=okta"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"org": nil,
			},
		},
	},
}
