// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

const (
	// Discovery flags
	DiscoveryOrganization       = "organization"
	DiscoveryFolders            = "folders"
	DiscoveryInstances          = "instances"
	DiscoveryProjects           = "projects"
	DiscoveryComputeImages      = "compute-images"
	DiscoveryComputeNetworks    = "compute-networks"
	DiscoveryComputeSubnetworks = "compute-subnetworks"
	DiscoveryComputeFirewalls   = "compute-firewalls"
	DiscoveryGkeClusters        = "gke-clusters"
	DiscoveryStorageBuckets     = "storage-buckets"
	DiscoveryBigQueryDatasets   = "bigquery-datasets"
)

var Config = plugin.Provider{
	Name:    "gcp",
	ID:      "go.mondoo.com/cnquery/providers/gcp",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:  "gcp",
			Use:   "gcp",
			Short: "GCP Cloud",
			Discovery: []string{
				DiscoveryOrganization,
				DiscoveryFolders,
				DiscoveryInstances,
				DiscoveryProjects,
				DiscoveryComputeImages,
				DiscoveryComputeNetworks,
				DiscoveryComputeSubnetworks,
				DiscoveryComputeFirewalls,
				DiscoveryGkeClusters,
				DiscoveryStorageBuckets,
				DiscoveryBigQueryDatasets,
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
