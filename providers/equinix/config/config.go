// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/equinix/provider"
)

var Config = plugin.Provider{
	Name:            "equinix",
	ID:              "go.mondoo.com/cnquery/providers/equinix",
	Version:         "9.1.16",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "equinix",
			Use:   "equinix [org <org id>] [project <project-id>] [--token <token>]",
			Short: "an Equinix Metal organization",
			Long: `equinix is designed for querying resources within a specified 
project or organization on the Equinix Metal platform.

Available Commands:
  org <org id>              Specifies the organization to interact with, using the organization identifier.
  project <project-id>      Specifies the project to interact with, using the project identifier.

If the PACKET_AUTH_TOKEN environment variable is set, the token flag is not required.
`,
			MinArgs:   2,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    " Specifies the Equinix API token for authenticating the user",
				},
			},
		},
	},
}
