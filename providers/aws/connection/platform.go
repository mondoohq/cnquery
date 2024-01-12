// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import "go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"

func (a *AwsConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject(a.PlatformOverride)
}

func GetPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "aws" && platformName != "" {
		return &inventory.Platform{
			Name:    platformName,
			Title:   getTitleForPlatformName(platformName),
			Kind:    "aws-object",
			Runtime: "aws",
		}
	}
	return &inventory.Platform{
		Name:    "aws",
		Title:   "AWS Account",
		Kind:    "api",
		Runtime: "aws",
	}
}

func getTitleForPlatformName(name string) string {
	switch name {
	case "aws-s3-bucket":
		return "AWS S3 Bucket"
	case "aws-cloudtrail-trail":
		return "AWS CloudTrail Trail"
	case "aws-rds-dbinstance":
		return "AWS RDS DB Instance"
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
