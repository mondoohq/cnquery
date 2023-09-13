// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

// Discovery Flags
const (
	DiscoveryInstances    = "instances"
	DiscoverySSMInstances = "ssm-instances"
	DiscoveryECR          = "ecr"
	DiscoveryECS          = "ecs"

	DiscoveryAll  = "all"  // resources, accounts, instances, ecr, ecs, everything
	DiscoveryAuto = "auto" // just the account for now

	// API scan
	DiscoveryAccounts                   = "accounts"
	DiscoveryResources                  = "resources"          // all the resources
	DiscoveryECSContainersAPI           = "ecs-containers-api" // need dedup story
	DiscoveryECRImageAPI                = "ecr-image-api"      // need policy + dedup story
	DiscoveryEC2InstanceAPI             = "ec2-instances-api"  // need policy + dedup story
	DiscoverySSMInstanceAPI             = "ssm-instances-api"  // need policy + dedup story
	DiscoveryS3Buckets                  = "s3-buckets"
	DiscoveryCloudtrailTrails           = "cloudtrail-trails"
	DiscoveryRdsDbInstances             = "rds-dbinstances"
	DiscoveryVPCs                       = "vpcs"
	DiscoverySecurityGroups             = "security-groups"
	DiscoveryIAMUsers                   = "iam-users"
	DiscoveryIAMGroups                  = "iam-groups"
	DiscoveryCloudwatchLoggroups        = "cloudwatch-loggroups"
	DiscoveryLambdaFunctions            = "lambda-functions"
	DiscoveryDynamoDBTables             = "dynamodb-tables"
	DiscoveryRedshiftClusters           = "redshift-clusters"
	DiscoveryVolumes                    = "ec2-volumes"
	DiscoverySnapshots                  = "ec2-snapshots"
	DiscoveryEFSFilesystems             = "efs-filesystems"
	DiscoveryAPIGatewayRestAPIs         = "gateway-restapis"
	DiscoveryELBLoadBalancers           = "elb-loadbalancers"
	DiscoveryESDomains                  = "es-domains"
	DiscoveryKMSKeys                    = "kms-keys"
	DiscoverySagemakerNotebookInstances = "sagemaker-notebookinstances"
)

var All = []string{
	DiscoveryAccounts,
	DiscoveryInstances,
	DiscoverySSMInstances,
	DiscoveryECR,
	DiscoveryECS,
}

var Auto = []string{
	DiscoveryAccounts,
}

var AllAPIResources = []string{
	// DiscoveryECSContainersAPI,
	// DiscoveryECRImageAPI,
	// DiscoveryEC2InstanceAPI,
	// DiscoverySSMInstanceAPI,
	DiscoveryS3Buckets,
	DiscoveryCloudtrailTrails,
	DiscoveryRdsDbInstances,
	DiscoveryVPCs,
	DiscoverySecurityGroups,
	DiscoveryIAMUsers,
	DiscoveryIAMGroups,
	DiscoveryCloudwatchLoggroups,
	DiscoveryLambdaFunctions,
	DiscoveryDynamoDBTables,
	DiscoveryRedshiftClusters,
	DiscoveryVolumes,
	DiscoverySnapshots,
	DiscoveryEFSFilesystems,
	DiscoveryAPIGatewayRestAPIs,
	DiscoveryELBLoadBalancers,
	DiscoveryESDomains,
	DiscoveryKMSKeys,
	DiscoverySagemakerNotebookInstances,
}
