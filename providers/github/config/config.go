// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/github/provider"
)

var Config = plugin.Provider{
	Name:            "github",
	ID:              "go.mondoo.com/cnquery/v9/providers/github",
	Version:         "9.1.14",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "github",
			Use:       "github",
			Short:     "a GitHub organization or repository",
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
