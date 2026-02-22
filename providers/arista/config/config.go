// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/arista/provider"
)

var Config = plugin.Provider{
	Name:            "arista",
	ID:              "go.mondoo.com/cnquery/v9/providers/arista",
	Version:         "11.1.3",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "arista",
			Use:   "arista user@host",
			Short: "an Arista EOS device",
			Long: `Use the arista provider to query resources on an Arista EOS network device, including system information, interfaces, and configuration.

Examples:
  cnquery shell arista <user@host>
  cnspec scan arista <user@host> --ask-pass
`,
			Discovery: []string{},
			MinArgs:   1,
			MaxArgs:   1,
			Flags: []plugin.Flag{
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for connection password",
					ConfigEntry: "-",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=arista"},
		},
	},
}
