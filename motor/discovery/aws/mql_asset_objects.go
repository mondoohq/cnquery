package aws

import (
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
)

func getTitleFamily(awsObject awsObject) (awsObjectPlatformInfo, error) {
	switch awsObject.service {
	case "s3":
		if awsObject.objectType == "bucket" {
			return awsObjectPlatformInfo{title: "AWS S3 Bucket", platform: "aws-s3-bucket"}, nil
		}
	case "cloudtrail":
		if awsObject.objectType == "trail" {
			return awsObjectPlatformInfo{title: "AWS Cloudtrail Trail", platform: "aws-cloudtrail-trail"}, nil
		}
	case "rds":
		if awsObject.objectType == "dbinstance" {
			return awsObjectPlatformInfo{title: "AWS RDS DBInstance", platform: "aws-rds-dbinstance"}, nil
		}
	case "dynamodb":
		if awsObject.objectType == "table" {
			return awsObjectPlatformInfo{title: "AWS DynamoDB Table", platform: "aws-dynamodb-table"}, nil
		}
	case "redshift":
		if awsObject.objectType == "cluster" {
			return awsObjectPlatformInfo{title: "AWS Redshift Cluster", platform: "aws-redshift-cluster"}, nil
		}
	case "vpc":
		if awsObject.objectType == "vpc" {
			return awsObjectPlatformInfo{title: "AWS VPC", platform: "aws-vpc"}, nil
		}
	case "ec2":
		switch awsObject.objectType {
		case "securitygroup":
			return awsObjectPlatformInfo{title: "AWS Security Group", platform: "aws-security-group"}, nil
		case "volume":
			return awsObjectPlatformInfo{title: "AWS EC2 Volume", platform: "aws-ec2-volume"}, nil
		case "snapshot":
			return awsObjectPlatformInfo{title: "AWS EC2 Snapshot", platform: "aws-ec2-snapshot"}, nil
		}
	case "iam":
		switch awsObject.objectType {
		case "user":
			return awsObjectPlatformInfo{title: "AWS IAM User", platform: "aws-iam-user"}, nil

		case "group":
			return awsObjectPlatformInfo{title: "AWS IAM Group", platform: "aws-iam-group"}, nil
		}
	case "cloudwatch":
		if awsObject.objectType == "loggroup" {
			return awsObjectPlatformInfo{title: "AWS Cloudwatch LogGroup", platform: "aws-cloudwatch-loggroup"}, nil
		}
	case "lambda":
		if awsObject.objectType == "function" {
			return awsObjectPlatformInfo{title: "AWS Lambda Function", platform: "aws-lambda-function"}, nil
		}
	case "ecs":
		if awsObject.objectType == "container" {
			return awsObjectPlatformInfo{title: "AWS ECS Container", platform: "aws-ecs-container"}, nil
		}
	}
	return awsObjectPlatformInfo{}, errors.Newf("missing runtime info for aws object service %s type %s", awsObject.service, awsObject.objectType)
}

func s3Buckets(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	buckets := m.GetList("return aws.s3.buckets { arn name location tags }") // no id field
	for i := range buckets {
		b := buckets[i].(map[string]interface{})
		name := b["name"].(string)
		arn := b["arn"].(string)
		tags := b["tags"].(map[string]interface{})
		region := b["location"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "s3", objectType: "bucket",
				},
			}, tc))
	}
	return assets
}

func cloudtrailTrails(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	trails := m.GetList("return aws.cloudtrail.trails { arn name region }") // no id field
	for i := range trails {
		t := trails[i].(map[string]interface{})
		name := t["name"].(string)
		region := t["region"].(string)
		arn := t["arn"].(string)

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "cloudtrail", objectType: "trail",
				},
			}, tc))
	}
	return assets
}

func rdsInstances(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	rdsinstances := m.GetList("return aws.rds.dbInstances { id arn name tags region }")
	for i := range rdsinstances {
		r := rdsinstances[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		id := r["id"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: id, service: "rds", objectType: "dbinstance",
				},
			}, tc))
	}
	return assets
}

func vpcs(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	vpcs := m.GetList("return aws.vpcs { id arn region tags }")
	for i := range vpcs {
		r := vpcs[i].(map[string]interface{})
		arn := r["arn"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		id := r["id"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: id, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: id, service: "vpc", objectType: "vpc",
				},
			}, tc))
	}
	return assets
}

func securityGroups(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	securitygroups := m.GetList("return aws.ec2.securityGroups { id arn region tags name description }")
	for i := range securitygroups {
		r := securitygroups[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		id := r["id"].(string)
		description := r["description"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}
		tags["description"] = description

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: id, service: "ec2", objectType: "securitygroup",
				},
			}, tc))
	}
	return assets
}

func iamUsers(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	users := m.GetList("return aws.iam.users { id arn tags name }")
	for i := range users {
		r := users[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		id := r["id"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: "us-east-1", arn: arn,
					id: id, service: "iam", objectType: "user",
				},
			}, tc))
	}
	return assets
}

func iamGroups(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	users := m.GetList("return aws.iam.groups { id arn name usernames }")
	for i := range users {
		r := users[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		usernames := r["usernames"].([]string)
		id := r["id"].(string)
		stringLabels := map[string]string{"usernames": strings.Join(usernames, ",")}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: "us-east-1", arn: arn,
					id: id, service: "iam", objectType: "group",
				},
			}, tc))
	}
	return assets
}

func cloudwatchLoggroups(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	loggroups := m.GetList("return aws.cloudwatch.logGroups { arn name region }")
	for i := range loggroups {
		r := loggroups[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		region := r["region"].(string)
		stringLabels := make(map[string]string)

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "cloudwatch", objectType: "loggroup",
				},
			}, tc))
	}
	return assets
}

func lambdaFunctions(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	lambdafunctions := m.GetList("return aws.lambda.functions { arn name region tags }")
	for i := range lambdafunctions {
		r := lambdafunctions[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "lambda", objectType: "function",
				},
			}, tc))
	}
	return assets
}

func dynamodbTables(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	dynamodbtables := m.GetList("return aws.dynamodb.tables { arn name region tags }")
	for i := range dynamodbtables {
		r := dynamodbtables[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "dynamodb", objectType: "table",
				},
			}, tc))
	}

	globaltables := m.GetList("return aws.dynamodb.globalTables { arn name region tags }")
	for i := range globaltables {
		r := globaltables[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "dynamodb", objectType: "table",
				},
			}, tc))
	}
	return assets
}

func redshiftClusters(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	clusters := m.GetList("return aws.redshift.clusters { arn name region tags }")
	for i := range clusters {
		r := clusters[i].(map[string]interface{})
		arn := r["arn"].(string)
		name := r["name"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "redshift", objectType: "cluster",
				},
			}, tc))
	}
	return assets
}

func ec2Volumes(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	volumes := m.GetList("return aws.ec2.volumes { arn id region tags }")
	for i := range volumes {
		r := volumes[i].(map[string]interface{})
		arn := r["arn"].(string)
		id := r["id"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: id, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: id, service: "ec2", objectType: "volume",
				},
			}, tc))
	}
	return assets
}

func ec2Snapshots(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	snapshots := m.GetList("return aws.ec2.snapshots { arn id region tags }")
	for i := range snapshots {
		r := snapshots[i].(map[string]interface{})
		arn := r["arn"].(string)
		id := r["id"].(string)
		tags := r["tags"].(map[string]interface{})
		region := r["region"].(string)
		stringLabels := make(map[string]string)
		for k, v := range tags {
			stringLabels[k] = v.(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: id, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: id, service: "ec2", objectType: "snapshot",
				},
			}, tc))
	}
	return assets
}

func ecsContainers(m *MqlDiscovery, account string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}

	containers := m.GetList("return aws.ecs.containers { arn taskDefinitionArn name publicIp image region }")
	for i := range containers {
		c := containers[i].(map[string]interface{})
		arn := c["arn"].(string)
		name := c["name"].(string)
		publicIp := c["publicIp"].(string)
		image := c["image"].(string)
		region := c["region"].(string)
		taskDefArn := c["taskDefinitionArn"].(string)
		stringLabels := map[string]string{common.IPLabel: publicIp, "image": image, "taskDefinitionArn": taskDefArn}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: stringLabels,
				awsObject: awsObject{
					account: account, region: region, arn: arn,
					id: name, service: "ecs", objectType: "container",
				},
			}, tc))
	}
	return assets
}
