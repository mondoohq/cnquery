// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/oci/provider"
)

var Config = plugin.Provider{
	Name:            "oci",
	ID:              "go.mondoo.com/cnquery/providers/oci",
	Version:         "9.0.2",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "oci",
			Use:       "oci",
			Short:     "Oracle Cloud Infrastructure",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "tenancy",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The tenancy's OCID",
				},
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The user's OCID",
				},
				{
					Long:    "region",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The selected region",
				},
				{
					Long:    "key-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The path to the private key, that will be used for authentication",
				},
				{
					Long:    "fingerprint",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The fingerprint of the private key",
				},
				{
					Long:    "key-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The passphrase for private key, that will be used for authentication",
				},
			},
		},
	},
}
