// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

func (a *AwsConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject(a.PlatformOverride, a.accountId)
}

func GetPlatformForObject(platformName string, accountId string) *inventory.Platform {
	if platformName != "aws" && platformName != "" {
		return &inventory.Platform{
			Name:                  platformName,
			Title:                 getTitleForPlatformName(platformName),
			Kind:                  getPlatformKind(platformName),
			Runtime:               "aws",
			TechnologyUrlSegments: getTechnologyUrlSegments(accountId, platformName),
		}
	}
	return &inventory.Platform{
		Name:                  "aws",
		Title:                 "AWS Account",
		Kind:                  "api",
		Runtime:               "aws",
		TechnologyUrlSegments: []string{"aws", accountId, "account"},
	}
}

func getPlatformKind(platformName string) string {
	switch platformName {
	case
		"aws-ebs-snapshot",
		// "aws-ecr-image",
		// "aws-ecs-container",
		"aws-ec2-instance",
		"aws-ssm-instance":
		return inventory.AssetKindCloudVM
	case "aws":
		return "api"
	default:
		return "aws-object"
	}
}

func getTechnologyUrlSegments(accountId string, platformName string) []string {
	return []string{"aws", accountId, getServiceName(platformName)}
}

func getServiceName(platformName string) string {
	switch platformName {
	case "aws-s3-bucket":
		return "s3"
	case "aws-cloudtrail-trail":
		return "cloudtrail"
	case "aws-rds-dbinstance":
		return "rds"
	case "aws-rds-dbcluster":
		return "rds"
	case "aws-dynamodb-table":
		return "dynamodb"
	case "aws-redshift-cluster":
		return "redshift"
	case "aws-vpc":
		return "vpc"
	case "aws-security-group":
		return "ec2"
	case "aws-ebs-volume":
		return "ec2"
	case "aws-ebs-snapshot":
		return "ec2"
	case "aws-iam-user":
		return "iam"
	case "aws-iam-group":
		return "iam"
	case "aws-cloudwatch-loggroup":
		return "cloudwatch"
	case "aws-lambda-function":
		return "lambda"
	case "aws-ecs-container":
		return "ecs"
	case "aws-efs-filesystem":
		return "efs"
	case "aws-gateway-restapi":
		return "apigateway"
	case "aws-elb-loadbalancer":
		return "elb"
	case "aws-es-domain":
		return "es"
	case "aws-kms-key":
		return "kms"
	case "aws-sagemaker-notebookinstance":
		return "sagemaker"
	case "aws-ec2-instance":
		return "ec2"
	case "aws-ssm-instance":
		return "ec2"
	case "aws-ecr-image":
		return "ecr"
	}
	return "other"
}

func getTitleForPlatformName(name string) string {
	switch name {
	case "aws-s3-bucket":
		return "AWS S3 Bucket"
	case "aws-cloudtrail-trail":
		return "AWS CloudTrail Trail"
	case "aws-rds-dbinstance":
		return "AWS RDS DB Instance"
	case "aws-rds-dbcluster":
		return "AWS RDS DB Cluster"
	case "aws-dynamodb-table":
		return "AWS DynamoDB Table"
	case "aws-redshift-cluster":
		return "AWS Redshift Cluster"
	case "aws-vpc":
		return "AWS VPC"
	case "aws-security-group":
		return "AWS Security Group"
	case "aws-ebs-volume":
		return "AWS EBS Volume"
	case "aws-ebs-snapshot":
		return "AWS EBS Snapshot"
	case "aws-iam-user":
		return "AWS IAM User"
	case "aws-iam-group":
		return "AWS IAM Group"
	case "aws-cloudwatch-loggroup":
		return "AWS CloudWatch Log Group"
	case "aws-lambda-function":
		return "AWS Lambda Function"
	case "aws-ecs-container":
		return "AWS ECS Container"
	case "aws-efs-filesystem":
		return "AWS EFS Filesystem"
	case "aws-gateway-restapi":
		return "AWS Gateway REST API"
	case "aws-elb-loadbalancer":
		return "AWS ELB Load Balancer"
	case "aws-es-domain":
		return "AWS ES Domain"
	case "aws-kms-key":
		return "AWS KMS Key"
	case "aws-sagemaker-notebookinstance":
		return "AWS SageMaker Notebook Instance"
	case "aws-ec2-instance":
		return "AWS EC2 Instance"
	case "aws-ssm-instance":
		return "AWS SSM Instance"
	case "aws-ecr-image":
		return "AWS ECR Image"
	}
	return "Amazon Web Services"
}
