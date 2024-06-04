// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/opcua/provider"
)

var Config = plugin.Provider{
	Name:            "opcua",
	ID:              "go.mondoo.com/cnquery/v9/providers/opcua",
	Version:         "11.0.13",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "opcua",
			Use:   "opcua [--endpoint <endpoint>]",
			Short: "an OPC UA device",
			Long: `opcua is designed for querying resources on an OPC UA (Open Platform 
Communications Unified Architecture) server, a protocol facilitating machine-to-machine communications within 
the realm of industrial automation.
`,

			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "endpoint",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "OPC UA endpoint URL of the OPC UA server in the format opc.tcp://<host>:<port>",
					Option:  plugin.FlagOption_Required,
				},
			},
		},
	},
}
