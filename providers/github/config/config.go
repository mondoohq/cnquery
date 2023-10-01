// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/github/provider"
)

var Config = plugin.Provider{
	Name:            "github",
	ID:              "go.mondoo.com/cnquery/providers/github",
	Version:         "9.0.3",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "github",
			Use:       "github",
			Short:     "GitHub",
			MinArgs:   2,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide GitHub personal access token.",
				},
			},
		},
	},
}
