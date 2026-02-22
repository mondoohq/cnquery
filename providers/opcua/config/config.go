// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/opcua/provider"
)

var Config = plugin.Provider{
	Name:            "opcua",
	ID:              "go.mondoo.com/cnquery/v9/providers/opcua",
	Version:         "11.0.136",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "opcua",
			Use:   "opcua [--endpoint <endpoint>]",
			Short: "an OPC UA device",
			Long: `Use the opcua provider to query resources on an Open Platform Communications Unified Architecture (OPC UA) server or device. OPC UA is a protocol for machine-to-machine communication in industrial automation.

Examples:
  cnquery shell opcua --endpoint opc.tcp://<host>:<port>
  cnspec scan opcua --endpoint opc.tcp://<host>:<port>
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
