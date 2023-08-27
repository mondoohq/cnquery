// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "equinix",
	ID:      "go.mondoo.com/cnquery/providers/equinix",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "equinix",
			Use:       "equinix",
			Short:     "Equinix Metal",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Slack API token",
				},
			},
		},
	},
}
