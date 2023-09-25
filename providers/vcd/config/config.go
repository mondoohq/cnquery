// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/vcd/provider"
)

var Config = plugin.Provider{
	Name:            "vcd",
	ID:              "go.mondoo.com/cnquery/providers/vcd",
	Version:         "9.0.1",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "vcd",
			Use:       "vcd",
			Short:     "VMware Cloud Director",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "vCloud Director user",
				},
				{
					Long:    "host",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "vCloud Director user",
				},
				{
					Long:    "organization",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "vCloud Director Organization (optional)",
				},
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
}
