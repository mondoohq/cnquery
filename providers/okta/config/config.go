// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/okta/provider"
)

var Config = plugin.Provider{
	Name:            "okta",
	ID:              "go.mondoo.com/cnquery/v9/providers/okta",
	Version:         "10.0.2",
	ConnectionTypes: []string{provider.ConnectionType},
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
