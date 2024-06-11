// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ms365/provider"
)

var Config = plugin.Provider{
	Name:            "ms365",
	ID:              "go.mondoo.com/cnquery/v9/providers/ms365",
	Version:         "11.0.17",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "ms365",
			Use:       "ms365",
			Short:     "a Microsoft 365 account",
			MinArgs:   0,
			MaxArgs:   5,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "tenant-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Directory (tenant) ID of the service principal",
				},
				{
					Long:    "client-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Application (client) ID of the service principal",
				},
				{
					Long:    "organization",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The organization to be scanned",
				},
				{
					Long:    "sharepoint-url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The sharepoint URL to be scanned",
				},
				{
					Long:    "client-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Secret for application",
				},
				{
					Long:    "certificate-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path (in PKCS #12/PFX or PEM format) to the authentication certificate",
				},
				{
					Long:    "certificate-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Passphrase for the authentication certificate file",
				},
			},
		},
	},
}
