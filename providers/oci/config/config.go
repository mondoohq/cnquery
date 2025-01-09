// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/oci/provider"
)

var Config = plugin.Provider{
	Name:            "oci",
	ID:              "go.mondoo.com/cnquery/v9/providers/oci",
	Version:         "11.0.50",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "oci",
			Use:       "oci",
			Short:     "an Oracle Cloud Infrastructure tenancy",
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
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=oci"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"tenancy": nil,
			},
		},
	},
}
