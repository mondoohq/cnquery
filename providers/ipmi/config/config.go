// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/ipmi/provider"
)

var Config = plugin.Provider{
	Name:            "ipmi",
	ID:              "go.mondoo.com/cnquery/v9/providers/ipmi",
	Version:         "10.4.1",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ipmi",
			Use:   "ipmi user@host",
			Short: "an IPMI interface",
			Long: `ipmi is designed for querying resources via the Intelligent Platform Management Interface (IPMI).
IPMI provides management and monitoring capabilities  independently of the host system's CPU,
firmware (BIOS or UEFI), and operating system.
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{provider.ConnectionType},
			Flags: []plugin.Flag{
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for connection password.",
					ConfigEntry: "-",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password for IPMI connection.",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
			},
		},
	},
}
