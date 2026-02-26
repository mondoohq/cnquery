// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/oci/provider"
)

var Config = plugin.Provider{
	Name:            "oci",
	ID:              "go.mondoo.com/cnquery/v9/providers/oci",
	Version:         "11.1.5",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "oci",
			Use:   "oci",
			Short: "an Oracle Cloud Infrastructure tenancy",
			Long: `Use the oci provider to query resources in an Oracle Cloud Infrastructure tenancy, including compute instances, networks, storage, and identity resources.

Examples:
  cnquery shell oci --tenancy <tenancy_ocid> --user <user_ocid> --region <region> --key-path <path_to_private_key> --fingerprint <key_fingerprint>
  cnspec scan oci --tenancy <tenancy_ocid> --user <user_ocid> --region <region> --key-path <path_to_private_key> --fingerprint <key_fingerprint>
`,
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
					Desc:    "The OCI region to connect to (e.g., us-ashburn-1)",
				},
				{
					Long:    "key-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to the private key file for API key authentication",
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
					Desc:    "Passphrase for the private key file, if encrypted",
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
