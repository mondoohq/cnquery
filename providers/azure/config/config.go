// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

// Discovery flags
const (
	DiscoveryAuto          = "auto"
	DiscoveryAll           = "all"
	DiscoverySubscriptions = "subscriptions"
	DiscoveryInstances     = "instances"
	// TODO: this probably needs some more work on the linking to its OS counterpart side
	DiscoveryInstancesApi      = "instances-api"
	DiscoverySqlServers        = "sql-servers"
	DiscoveryPostgresServers   = "postgres-servers"
	DiscoveryMySqlServers      = "mysql-servers"
	DiscoveryMariaDbServers    = "mariadb-servers"
	DiscoveryStorageAccounts   = "storage-accounts"
	DiscoveryStorageContainers = "storage-containers"
	DiscoveryKeyVaults         = "keyvaults-vaults"
	DiscoverySecurityGroups    = "security-groups"
)

var Config = plugin.Provider{
	Name:    "azure",
	ID:      "go.mondoo.com/cnquery/providers/azure",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:    "azure",
			Use:     "azure",
			Short:   "azure",
			MinArgs: 0,
			MaxArgs: 8,
			Discovery: []string{
				DiscoveryAuto,
				DiscoveryAll,
				DiscoverySubscriptions,
				DiscoveryInstances,
				DiscoveryInstancesApi,
				DiscoverySqlServers,
				DiscoveryPostgresServers,
				DiscoveryMySqlServers,
				DiscoveryMariaDbServers,
				DiscoveryStorageAccounts,
				DiscoveryStorageContainers,
				DiscoveryKeyVaults,
				DiscoverySecurityGroups,
			},
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
				{
					Long:    "subscription",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "ID of the Azure subscription to scan.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscriptions",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to include.",
					Option:  plugin.FlagOption_Hidden,
				},
				{
					Long:    "subscriptions-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Comma-separated list of Azure subscriptions to exclude.",
					Option:  plugin.FlagOption_Hidden,
				},
			},
		},
	},
}
