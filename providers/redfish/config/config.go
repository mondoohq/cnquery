// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/redfish/provider"
)

var Config = plugin.Provider{
	Name:            "redfish",
	ID:              "go.mondoo.com/mql/providers/redfish",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "redfish",
			Use:   "redfish USER@HOST",
			Short: "a Redfish management controller (BMC)",
			Long: `Use the redfish provider to query a server's baseboard management controller (BMC)
over the DMTF Redfish REST API, including HPE iLO and Dell iDRAC.

Redfish provides out-of-band management and inventory independently of the host
system's CPU, firmware (BIOS or UEFI), and operating system.

Examples:
  cnspec shell redfish <user@host> --ask-pass
  cnspec scan redfish <user@host> --password <password> --insecure
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
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
					Desc:        "Set the connection password for the Redfish controller",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:        "insecure",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Do not verify the controller's TLS certificate (BMCs ship self-signed certificates by default)",
					ConfigEntry: "-",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=redfish"},
		},
	},
}
