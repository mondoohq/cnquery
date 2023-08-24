// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "opcua",
	ID:      "go.mondoo.com/cnquery/providers/opcua",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "opcua",
			Use:       "opcua",
			Short:     "OPC UA",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "endpoint",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "OPC UA service endpoint",
				},
			},
		},
	},
}
