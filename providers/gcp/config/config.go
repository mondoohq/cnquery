// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/gcp/connection/gcpinstancesnapshot"
	"go.mondoo.com/cnquery/v12/providers/gcp/provider"
	"go.mondoo.com/cnquery/v12/providers/gcp/resources"
)

var Config = plugin.Provider{
	Name:    "gcp",
	ID:      "go.mondoo.com/cnquery/v9/providers/gcp",
	Version: "11.0.124",
	ConnectionTypes: []string{
		provider.ConnectionType,
		string(gcpinstancesnapshot.SnapshotConnectionType),
	},
	Connectors: []plugin.Connector{
		{
			Name:  "gcp",
			Use:   "gcp",
			Short: "a Google Cloud project or folder",
			Long: `Use the gcp provider to query resources within Google Cloud Platform (GCP), including databases, services, instances, containers, and more.

Examples without logging into and configuring GCP:
  cnquery shell gcp org <ORGANIZATION-ID> --credentials-path <PATH-TO-YOUR-SERVICE-ACCT>
  cnspec scan gcp project <PROJECT-ID> --credentials-path <PATH-TO-YOUR-SERVICE-ACCT>

Note:
  If you log into GCP and configure the project you want to query or scan, you can omit credentials. To learn how, read https://mondoo.com/docs/cnspec/cloud/gcp/.

Examples with the GCP project configured:
  cnspec scan gcp folder <FOLDER-ID>
  cnquery shell gcp project
`,
			MaxArgs: 2,
			Discovery: []string{
				resources.DiscoveryOrganization,
				resources.DiscoveryFolders,
				resources.DiscoveryComputeInstances,
				resources.DiscoveryProjects,
				resources.DiscoveryComputeImages,
				resources.DiscoveryComputeNetworks,
				resources.DiscoveryComputeSubnetworks,
				resources.DiscoveryComputeFirewalls,
				resources.DiscoveryGkeClusters,
				resources.DiscoveryStorageBuckets,
				resources.DiscoveryBigQueryDatasets,
				resources.DiscoverCloudSQLMySQL,
				resources.DiscoverCloudSQLPostgreSQL,
				resources.DiscoverCloudSQLSQLServer,
				resources.DiscoverCloudDNSZones,
				resources.DiscoverCloudKMSKeyrings,
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
					Desc:    "[gcp gcr] specify the GCR repository to scan",
				},
				{
					Long:    "project-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "[gcp snapshot, gcp instance] specify the GCP project ID where the target instance is located",
				},
				{
					Long:    "zone",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "[gcp snapshot, gcp instance] specify the GCP zone where the target instance is located",
				},
				{
					Long:    "create-snapshot",
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "[gcp snapshot, gcp instance] create a new snapshot instead of using the latest available snapshot (only used for instance)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=gcp"},
			Key:          "project",
			Title:        "Project",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": {
					Key:   "service",
					Title: "Service",
					Values: map[string]*inventory.AssetUrlBranch{
						"project": nil,
						"compute": {
							Key:   "region",
							Title: "Region",
							Values: map[string]*inventory.AssetUrlBranch{
								"*": {
									Key:   "object",
									Title: "Compute Object",
									Values: map[string]*inventory.AssetUrlBranch{
										"instance": {
											Key: "type",
											Values: map[string]*inventory.AssetUrlBranch{
												"resource": nil,
												// os ... references the os asset tree
											},
										},
										"image":      nil,
										"network":    nil,
										"subnetwork": nil,
										"other":      nil,
									},
								},
							},
						},
						"storage": {
							Key:   "region",
							Title: "Region",
							Values: map[string]*inventory.AssetUrlBranch{
								"*": {
									Key:   "object",
									Title: "Storage Object",
									Values: map[string]*inventory.AssetUrlBranch{
										"bucket": nil,
										"other":  nil,
									},
								},
							},
						},
						"gke": {
							Key:   "region",
							Title: "Region",
							Values: map[string]*inventory.AssetUrlBranch{
								"*": {
									Key:   "object",
									Title: "GKE Object",
									Values: map[string]*inventory.AssetUrlBranch{
										"cluster": nil,
										"other":   nil,
									},
								},
							},
						},
						"other": nil,
					},
				},
			},
		},
	},
}
