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
	Version:         "11.1.27",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "mondoo",
			Use:   "mondoo",
			Short: "Mondoo Platform",
			Long: `Use the mondoo provider to query resources in Mondoo Platform.

To query Mondoo Platform from a workstation, the workstation must be registered with Mondoo Platform. To learn how to register a workstation, read https://mondoo.com/docs/cnspec/cnspec-adv-install/registration/. 

Examples:
  cnquery shell mondoo
	cnspec scan mondoo
`,
			MinArgs:   0,
			MaxArgs:   4,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
