// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/nmap/provider"
)

var Config = plugin.Provider{
	Name:            "nmap",
	ID:              "go.mondoo.com/cnquery/v11/providers/nmap",
	Version:         "11.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "nmap",
			Use:       "nmap",
			Short:     "a Nmap network scanner",
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
