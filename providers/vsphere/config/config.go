// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/vsphere/provider"
	"go.mondoo.com/cnquery/v10/providers/vsphere/resources"
)

var Config = plugin.Provider{
	Name:            "vsphere",
	ID:              "go.mondoo.com/cnquery/v9/providers/vsphere",
	Version:         "10.2.1",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "vsphere",
			Use:   "vsphere user@host",
			Short: "a VMware vSphere installation",
			Discovery: []string{
				resources.DiscoveryApi,
				resources.DiscoveryInstances,
				resources.DiscoveryHostMachines,
			},
			MinArgs: 1,
			MaxArgs: 1,
			Flags: []plugin.Flag{
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for connection password",
					ConfigEntry: "-",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
			},
		},
	},
}
