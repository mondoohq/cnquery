// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "okta",
	ID:      "go.mondoo.com/cnquery/providers/okta",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "okta",
			Use:       "okta",
			Short:     "Okta",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "organization",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Specify the Okta organization to scan",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Okta access token",
				},
			},
		},
	},
}
