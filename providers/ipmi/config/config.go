// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ipmi/provider"
)

var Config = plugin.Provider{
	Name:            "ipmi",
	ID:              "go.mondoo.com/cnquery/v9/providers/ipmi",
	Version:         "11.0.34",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ipmi",
			Use:   "ipmi USER@HOST",
			Short: "an IPMI interface",
			Long: `Use the ipmi provider to query resources using the Intelligent Platform Management Interface (IPMI).

IPMI provides management and monitoring capabilities independently of the host system's CPU,
firmware (BIOS or UEFI), and operating system.

Examples:
  cnquery shell ipmi <user@host>
	cnspec scan ipmi <user@host>
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{provider.ConnectionType},
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
					Desc:        "Set the connection password for IPMI connection",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=ipmi"},
		},
	},
}
