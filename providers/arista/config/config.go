// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/arista/provider"
)

var Config = plugin.Provider{
	Name:            "arista",
	ID:              "go.mondoo.com/cnquery/v9/providers/arista",
	Version:         "11.1.4",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "arista",
			Use:   "arista user@host",
			Short: "an Arista EOS device",
			Long: `Use the arista provider to query resources on an Arista EOS network device, including system information, interfaces, and configuration.

Examples:
  cnquery shell arista <user@host> --pasword <password>
  cnspec scan arista <user@host> --ask-pass

Note: The arista provider uses requires access to the Arista API over HTTPS. You may be able to SSH to a device, but not access the API. To view the status of the API run 'show management api http-commands' on the device.
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
