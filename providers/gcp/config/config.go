// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/gcp/connection/gcpinstancesnapshot"
	"go.mondoo.com/cnquery/providers/gcp/provider"
	"go.mondoo.com/cnquery/providers/gcp/resources"
)

var Config = plugin.Provider{
	Name:    "gcp",
	ID:      "go.mondoo.com/cnquery/providers/gcp",
	Version: "9.0.0",
	ConnectionTypes: []string{
		provider.ConnectionType,
		string(gcpinstancesnapshot.SnapshotConnectionType),
	},
	Connectors: []plugin.Connector{
		{
			Name:  "gcp",
			Use:   "gcp",
			Short: "GCP Cloud",
			Discovery: []string{
				resources.DiscoveryAll,
				resources.DiscoveryAuto,
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
				{
					Long:    "repository",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "specify the GCR repository to scan (only used for gcr sub command)",
				},
				{
					Long:    "project-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "specify the GCP project ID where the target instance is located (only used for snapshots)",
				},
				{
					Long:    "zone",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "specify the GCP zone where the target instance is located (only used for snapshots)",
				},
			},
		},
	},
}
