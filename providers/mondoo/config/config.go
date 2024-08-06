// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/mondoo/provider"
)

var Config = plugin.Provider{
	Name:            "mondoo",
	ID:              "go.mondoo.com/cnquery/v11/providers/mondoo",
	Version:         "11.1.2",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "mondoo",
			Use:       "mondoo",
			Short:     "Mondoo Platform",
			MinArgs:   0,
			MaxArgs:   4,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
