// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/azure/connection/azureinstancesnapshot"
	"go.mondoo.com/cnquery/v11/providers/azure/provider"
	"go.mondoo.com/cnquery/v11/providers/azure/resources"
)

var Config = plugin.Provider{
	Name:    "azure",
	ID:      "go.mondoo.com/cnquery/v9/providers/azure",
	Version: "11.3.70",
	ConnectionTypes: []string{
		provider.ConnectionType,
		string(azureinstancesnapshot.SnapshotConnectionType),
	},
	Connectors: []plugin.Connector{
		{
			Name:  "azure",
			Use:   "azure",
			Short: "an Azure subscription",
			Long: `Use the azure provider to query resources within Microsoft Azure, including storage, compute instances, snapshots, databases, and more.

Examples run in your shell:
  cnspec scan azure compute instance <name> --client-id <your-client-id> --tenant-id <your-tenant-id> --client-secret <your-client-secret-value>
  cnspec scan azure compute snapshot <name> --client-id <your-client-id> --tenant-id <your-tenant-id> --client-secret <your-client-secret-value>
	cnquery shell azure <name> --client-id <your-client-id> --tenant-id <your-tenant-id> --client-secret <your-client-secret-value>

Examples run in the Azure CLI:	
  cnquery shell azure
  cnspec scan azure --subscription <specific-subscription-id>
`,
			MinArgs: 0,
			MaxArgs: 8,
			Discovery: []string{
				resources.DiscoverySubscriptions,
				resources.DiscoveryInstances,
				resources.DiscoveryInstancesApi,
				resources.DiscoverySqlServers,
				resources.DiscoveryPostgresServers,
				resources.DiscoveryPostgresFlexibleServers,
				resources.DiscoveryMySqlServers,
				resources.DiscoveryMySqlFlexibleServers,
				resources.DiscoveryMariaDbServers,
				resources.DiscoveryStorageAccounts,
				resources.DiscoveryStorageContainers,
				resources.DiscoveryKeyVaults,
				resources.DiscoverySecurityGroups,
			},
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
				{
					Long:    "subscription",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "ID of the Azure subscription to scan",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscriptions",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to include",
				},
				{
					Long:    "subscriptions-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to exclude",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=azure"},
			Key:          "tenant",
			Title:        "Tenant",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": {
					Key:   "subscription",
					Title: "Subscription",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": {
							Key: "service",
							Values: map[string]*inventory.AssetUrlBranch{
								"account":    nil,
								"compute":    nil,
								"mysql":      nil,
								"postgresql": nil,
								"mariadb":    nil,
								"sql":        nil,
								"storage": {
									Key: "object",
									Values: map[string]*inventory.AssetUrlBranch{
										"account":   nil,
										"container": nil,
										"other":     nil,
									},
								},
								"network": {
									Key: "object",
									Values: map[string]*inventory.AssetUrlBranch{
										"security-group": nil,
										"other":          nil,
									},
								},
								"keyvault": nil,
							},
						},
					},
				},
			},
		},
	},
}
