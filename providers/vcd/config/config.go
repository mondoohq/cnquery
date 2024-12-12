// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vcd/provider"
)

var Config = plugin.Provider{
	Name:            "vcd",
	ID:              "go.mondoo.com/cnquery/v9/providers/vcd",
	Version:         "11.0.45",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "vcd",
			Use:   "vcd [--user <user>] [--host <host>] [--organization <organization>] [--ask-pass] [--password <password>]",
			Short: "a VMware Cloud Director installation",
			Long: `vcd is designed for querying resources within for a VMware Cloud Director environment. VMware's 
vCloud Director (vCD), a platform that facilitates the operation and management of virtual resources within
a multi-tenant cloud environment.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "vCloud Director user",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "host",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "vCloud Director host",
					Option:  plugin.FlagOption_Required,
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
