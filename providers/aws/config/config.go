// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/provider"
)

var Config = plugin.Provider{
	Name:            "aws",
	ID:              "go.mondoo.com/cnquery/providers/aws",
	Version:         "9.0.0",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:    "aws",
			Use:     "aws",
			Short:   "aws account",
			MinArgs: 0,
			MaxArgs: 0,
			Discovery: []string{
				connection.DiscoveryAccounts,
				connection.DiscoveryAll,
				connection.DiscoveryAuto,

				connection.DiscoveryInstances,
				connection.DiscoverySSMInstances,
				connection.DiscoveryECR,
				connection.DiscoveryECS,

				connection.DiscoveryResources,
				connection.DiscoveryS3Buckets,
				connection.DiscoveryCloudtrailTrails,
				connection.DiscoveryRdsDbInstances,
				connection.DiscoveryVPCs,
				connection.DiscoverySecurityGroups,
				connection.DiscoveryIAMUsers,
				connection.DiscoveryIAMGroups,
				connection.DiscoveryCloudwatchLoggroups,
				connection.DiscoveryLambdaFunctions,
				connection.DiscoveryDynamoDBTables,
				connection.DiscoveryRedshiftClusters,
				connection.DiscoveryVolumes,
				connection.DiscoverySnapshots,
				connection.DiscoveryEFSFilesystems,
				connection.DiscoveryAPIGatewayRestAPIs,
				connection.DiscoveryELBLoadBalancers,
				connection.DiscoveryESDomains,
				connection.DiscoveryKMSKeys,
				connection.DiscoverySagemakerNotebookInstances,
			},
			Flags: []plugin.Flag{
				{
					Long:    "profile",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Profile to use when reading from ~/.aws/credentials",
				},
				{
					Long:    "region",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Region to use for authentication with the API. Note: this does not limit the discovery to the region",
				},
				{
					Long:    "role",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "ARN of the role to use for authentication with the API",
				},
				{
					Long:    "endpoint-url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Endpoint URL override for authentication with the API",
				},
			},
		},
	},
}
