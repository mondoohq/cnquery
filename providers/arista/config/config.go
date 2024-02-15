// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/arista/provider"
)

var Config = plugin.Provider{
	Name:            "arista",
	ID:              "go.mondoo.com/cnquery/v9/providers/arista",
	Version:         "9.3.0",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "arista",
			Use:       "arista user@host",
			Short:     "an Arista EOS device",
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
}
