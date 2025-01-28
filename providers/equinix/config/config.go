// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/equinix/provider"
)

var Config = plugin.Provider{
	Name:            "equinix",
	ID:              "go.mondoo.com/cnquery/v9/providers/equinix",
	Version:         "11.0.55",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "equinix",
			Use:   "equinix [org <org id>] [project <project-id>] [--token <token>]",
			Short: "an Equinix Metal organization",
			Long: `Use the equinix provider to query resources in a specified
project or organization on the Equinix Metal platform.

Available commands:
  org <org id>              Specifies the organization to interact with, using the organization identifier.
  project <project-id>      Specifies the project to interact with, using the project identifier.

If you set the PACKET_AUTH_TOKEN environment variable, you can omit the token flag.
`,
			MinArgs:   2,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The Equinix API token for authenticating the user",
				},
			},
		},
	},
}
