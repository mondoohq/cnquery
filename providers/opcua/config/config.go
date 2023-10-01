// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/opcua/provider"
)

var Config = plugin.Provider{
	Name:            "opcua",
	ID:              "go.mondoo.com/cnquery/providers/opcua",
	Version:         "9.0.3",
	ConnectionTypes: []string{provider.ConnectionType},
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
