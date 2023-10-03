// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/ms365/provider"
)

var Config = plugin.Provider{
	Name:            "ms365",
	ID:              "go.mondoo.com/cnquery/providers/ms365",
	Version:         "9.0.4",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "ms365",
			Use:       "ms365",
			Short:     "ms365",
			MinArgs:   0,
			MaxArgs:   5,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "tenant-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Directory (tenant) ID of the service principal.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "client-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Application (client) ID of the service principal.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "client-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Secret for application.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "certificate-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path (in PKCS #12/PFX or PEM format) to the authentication certificate.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "certificate-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Passphrase for the authentication certificate file.",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
	},
}
