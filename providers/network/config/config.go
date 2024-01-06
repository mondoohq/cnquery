// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/network/provider"
)

var Config = plugin.Provider{
	Name:            "network",
	ID:              "go.mondoo.com/cnquery/providers/network",
	Version:         "9.1.8",
	ConnectionTypes: []string{provider.HostConnectionType},
	CrossProviderTypes: []string{
		"go.mondoo.com/cnquery/providers/os",
		"go.mondoo.com/cnquery/providers/k8s",
		"go.mondoo.com/cnquery/providers/aws",
	},
	Connectors: []plugin.Connector{
		{
			Name:      "host",
			Use:       "host HOST",
			Short:     "a remote host",
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "insecure",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Disable TLS/SSL verification.",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
	},
}
