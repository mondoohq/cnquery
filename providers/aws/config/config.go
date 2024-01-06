// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/aws/connection/awsec2ebsconn"
	"go.mondoo.com/cnquery/v9/providers/aws/provider"
	"go.mondoo.com/cnquery/v9/providers/aws/resources"
)

var Config = plugin.Provider{
	Name:            "aws",
	ID:              "go.mondoo.com/cnquery/providers/aws",
	Version:         "9.1.25",
	ConnectionTypes: []string{provider.DefaultConnectionType, string(awsec2ebsconn.EBSConnectionType)},
	Connectors: []plugin.Connector{
		{
			Name:    "aws",
			Use:     "aws",
			Short:   "an AWS account",
			MinArgs: 0,
			MaxArgs: 4,
			Discovery: []string{
				resources.DiscoveryAccounts,
				resources.DiscoveryAll,
				resources.DiscoveryAuto,

				resources.DiscoveryInstances,
				resources.DiscoveryEC2InstanceAPI,
				resources.DiscoverySSMInstances,
				resources.DiscoverySSMInstanceAPI,
				resources.DiscoveryECR,
				resources.DiscoveryECS,

				resources.DiscoveryResources,
				resources.DiscoveryS3Buckets,
				resources.DiscoveryCloudtrailTrails,
				resources.DiscoveryRdsDbInstances,
				resources.DiscoveryVPCs,
				resources.DiscoverySecurityGroups,
				resources.DiscoveryIAMUsers,
				resources.DiscoveryIAMGroups,
				resources.DiscoveryCloudwatchLoggroups,
				resources.DiscoveryLambdaFunctions,
				resources.DiscoveryDynamoDBTables,
				resources.DiscoveryRedshiftClusters,
				resources.DiscoveryVolumes,
				resources.DiscoverySnapshots,
				resources.DiscoveryEFSFilesystems,
				resources.DiscoveryAPIGatewayRestAPIs,
				resources.DiscoveryELBLoadBalancers,
				resources.DiscoveryESDomains,
				resources.DiscoveryKMSKeys,
				resources.DiscoverySagemakerNotebookInstances,
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
				{
					Long:    "no-setup",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Override option for EBS scanning that tells it to not create the snapshot or volume",
				},
				{
					Long:    "scope",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Set Scope for the aws wafv2 either CLOUDFRONT or REGIONAL",
				},
				{
					Long:    "filters",
					Type:    plugin.FlagType_KeyValue,
					Default: "",
					Desc:    "Filter options",
				},
			},
		},
	},
}
