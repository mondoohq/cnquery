// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "go.mondoo.com/mql/v13/providers-sdk/v1/plugin"

// Platforms is the static catalog of platforms the AWS provider can emit via
// GetPlatformForObject: the account root plus one entry per discoverable AWS
// object type. Each object is an "aws-object" running in the "aws" runtime;
// the account root is an "api" platform. This is the single source of truth for
// both the provider config (config.Config.Platforms) and the runtime builder.
var Platforms = []*plugin.PlatformInfo{
	{Name: "aws", Title: "AWS Account", Kind: []string{"api"}, Runtime: []string{"aws"}},
	{Name: "aws-s3-bucket", Title: "AWS S3 Bucket", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-cloudtrail-trail", Title: "AWS CloudTrail Trail", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-rds-dbinstance", Title: "AWS RDS DB Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-rds-dbcluster", Title: "AWS RDS DB Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-dynamodb-table", Title: "AWS DynamoDB Table", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-redshift-cluster", Title: "AWS Redshift Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-vpc", Title: "AWS VPC", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-security-group", Title: "AWS Security Group", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ebs-volume", Title: "AWS EBS Volume", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ebs-snapshot", Title: "AWS EBS Snapshot", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-iam-user", Title: "AWS IAM User", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-iam-group", Title: "AWS IAM Group", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-cloudwatch-loggroup", Title: "AWS CloudWatch Log Group", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-lambda-function", Title: "AWS Lambda Function", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ecs-container", Title: "AWS ECS Container", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ecs-instance", Title: "AWS ECS Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-efs-filesystem", Title: "AWS EFS Filesystem", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-gateway-restapi", Title: "AWS Gateway REST API", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-elb-loadbalancer", Title: "AWS ELB Load Balancer", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-es-domain", Title: "AWS ES Domain", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-opensearch-domain", Title: "AWS OpenSearch Domain", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-kms-key", Title: "AWS KMS Key", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-sagemaker-notebookinstance", Title: "AWS SageMaker Notebook Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-sagemaker-processingjob", Title: "AWS SageMaker Processing Job", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-sagemaker-trainingjob", Title: "AWS SageMaker Training Job", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ec2-instance", Title: "AWS EC2 Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ssm-instance", Title: "AWS SSM Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ecr-image", Title: "AWS ECR Image", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ecr-repository", Title: "AWS ECR Repository", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-ecs-taskdefinition", Title: "AWS ECS Task Definition", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-route53-hostedzone", Title: "AWS Route 53 Hosted Zone", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-msk-cluster", Title: "AWS MSK Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-mq-broker", Title: "AWS MQ Broker", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-eks-cluster", Title: "AWS EKS Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-secretsmanager-secret", Title: "AWS Secrets Manager Secret", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-elasticache-cluster", Title: "AWS ElastiCache Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-cloudfront-distribution", Title: "AWS CloudFront Distribution", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-neptune-cluster", Title: "AWS Neptune Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-emr-cluster", Title: "AWS EMR Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-documentdb-cluster", Title: "AWS DocumentDB Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-memorydb-cluster", Title: "AWS MemoryDB Cluster", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-codebuild-project", Title: "AWS CodeBuild Project", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-cognito-userpool", Title: "AWS Cognito User Pool", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-transfer-server", Title: "AWS Transfer Family Server", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-apigatewayv2-api", Title: "AWS API Gateway V2 API", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-athena-workgroup", Title: "AWS Athena Workgroup", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-appstream-fleet", Title: "AWS AppStream Fleet", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-batch-jobdefinition", Title: "AWS Batch Job Definition", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-directoryservice-directory", Title: "AWS Directory Service Directory", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
	{Name: "aws-documentdb-instance", Title: "AWS DocumentDB Instance", Kind: []string{"aws-object"}, Runtime: []string{"aws"}},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the static descriptor for a platform name, or nil if
// the name is not in the catalog.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}
