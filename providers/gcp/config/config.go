// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/gcp/provider"
	"go.mondoo.com/cnquery/providers/gcp/resources"
)

var Config = plugin.Provider{
	Name:            "gcp",
	ID:              "go.mondoo.com/cnquery/providers/gcp",
	Version:         "9.0.0",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "gcp",
			Use:   "gcp",
			Short: "GCP Cloud",
			Discovery: []string{
				resources.DiscoveryOrganization,
				resources.DiscoveryFolders,
				resources.DiscoveryInstances,
				resources.DiscoveryProjects,
				resources.DiscoveryComputeImages,
				resources.DiscoveryComputeNetworks,
				resources.DiscoveryComputeSubnetworks,
				resources.DiscoveryComputeFirewalls,
				resources.DiscoveryGkeClusters,
				resources.DiscoveryStorageBuckets,
				resources.DiscoveryBigQueryDatasets,
			},
			Flags: []plugin.Flag{
				{
					Long:    "credentials-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The path to the service account credentials to access the APIs with",
				},
			},
		},
	},
}
