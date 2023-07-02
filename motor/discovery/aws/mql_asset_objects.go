package aws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources/packs/aws"
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
		case "instance":
			return awsObjectPlatformInfo{title: "AWS EC2 Instance", platform: "aws-ec2-instance"}, nil
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
		if awsObject.objectType == "instance" {
			return awsObjectPlatformInfo{title: "AWS ECS Container Instance", platform: "aws-ecs-instance"}, nil
		}
	case "efs":
		if awsObject.objectType == "filesystem" {
			return awsObjectPlatformInfo{title: "AWS EFS Filesystem", platform: "aws-efs-filesystem"}, nil
		}
	case "gateway":
		if awsObject.objectType == "restapi" {
			return awsObjectPlatformInfo{title: "AWS Gateway RESTAPI", platform: "aws-gateway-restapi"}, nil
		}
	case "elb":
		if awsObject.objectType == "loadbalancer" {
			return awsObjectPlatformInfo{title: "AWS ELB LoadBalancer", platform: "aws-elb-loadbalancer"}, nil
		}
	case "es":
		if awsObject.objectType == "domain" {
			return awsObjectPlatformInfo{title: "AWS ES Domain", platform: "aws-es-domain"}, nil
		}
	case "kms":
		if awsObject.objectType == "key" {
			return awsObjectPlatformInfo{title: "AWS KMS Key", platform: "aws-kms-key"}, nil
		}
	case "sagemaker":
		if awsObject.objectType == "notebookinstance" {
			return awsObjectPlatformInfo{title: "AWS Sagemaker NotebookInstance", platform: "aws-sagemaker-notebookinstance"}, nil
		}
	case "ssm":
		if awsObject.objectType == "instance" {
			return awsObjectPlatformInfo{title: "AWS SSM Instance", platform: "aws-ssm-instance"}, nil
		}
	case "ecr":
		if awsObject.objectType == "image" {
			return awsObjectPlatformInfo{title: "AWS ECR Image", platform: "aws-ecr-image"}, nil
		}
	}
	return awsObjectPlatformInfo{}, errors.New(fmt.Sprintf("missing runtime info for aws object service %s type %s", awsObject.service, awsObject.objectType))
}

func s3Buckets(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn      string
		Name     string
		Location string
		Tags     map[string]string
	}
	buckets, err := GetList[data](m, "return aws.s3.buckets { arn name location tags }") // no id field
	if err != nil {
		return nil, err
	}
	for i := range buckets {
		b := buckets[i]
		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: b.Name, labels: b.Tags,
				awsObject: awsObject{
					account: account, region: b.Location, arn: b.Arn,
					id: b.Name, service: "s3", objectType: "bucket",
				},
			}, tc))
	}
	return assets, nil
}

func cloudtrailTrails(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Name   string
		Region string
	}
	trails, err := GetList[data](m, "return aws.cloudtrail.trails { arn name region }") // no id field
	if err != nil {
		return nil, err
	}
	for i := range trails {
		t := trails[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: t.Name,
				awsObject: awsObject{
					account: account, region: t.Region, arn: t.Arn,
					id: t.Name, service: "cloudtrail", objectType: "trail",
				},
			}, tc))
	}
	return assets, nil
}

func rdsInstances(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Id     string
		Arn    string
		Name   string
		Region string
		Tags   map[string]string
	}
	rdsinstances, err := GetList[data](m, "return aws.rds.dbInstances { id arn name tags region }")
	if err != nil {
		return nil, err
	}
	for i := range rdsinstances {
		r := rdsinstances[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: r.Name, labels: r.Tags,
				awsObject: awsObject{
					account: account, region: r.Region, arn: r.Arn,
					id: r.Id, service: "rds", objectType: "dbinstance",
				},
			}, tc))
	}
	return assets, nil
}

func vpcs(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Id     string
		Arn    string
		Region string
		Tags   map[string]string
	}
	vpcs, err := GetList[data](m, "return aws.vpcs { id arn region tags }")
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		v := vpcs[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: v.Id, labels: v.Tags,
				awsObject: awsObject{
					account: account, region: v.Region, arn: v.Arn,
					id: v.Id, service: "vpc", objectType: "vpc",
				},
			}, tc))
	}
	return assets, nil
}

func securityGroups(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Id          string
		Arn         string
		Region      string
		Tags        map[string]string
		Name        string
		Description string
	}
	securitygroups, err := GetList[data](m, "return aws.ec2.securityGroups { id arn region tags name description }")
	if err != nil {
		return nil, err
	}
	for i := range securitygroups {
		s := securitygroups[i]
		s.Tags["description"] = s.Description

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: s.Name, labels: s.Tags,
				awsObject: awsObject{
					account: account, region: s.Region, arn: s.Arn,
					id: s.Id, service: "ec2", objectType: "securitygroup",
				},
			}, tc))
	}
	return assets, nil
}

func iamUsers(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Id   string
		Arn  string
		Tags map[string]string
		Name string
	}
	users, err := GetList[data](m, "return aws.iam.users { id arn tags name }")
	if err != nil {
		return nil, err
	}
	for i := range users {
		u := users[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: u.Name, labels: u.Tags,
				awsObject: awsObject{
					account: account, region: "us-east-1", arn: u.Arn,
					id: u.Id, service: "iam", objectType: "user",
				},
			}, tc))
	}
	return assets, nil
}

func iamGroups(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Id        string
		Arn       string
		Name      string
		Usernames []string
	}
	groups, err := GetList[data](m, "return aws.iam.groups { id arn name usernames }")
	if err != nil {
		return nil, err
	}
	for i := range groups {
		g := groups[i]
		tags := map[string]string{"usernames": strings.Join(g.Usernames, ",")}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: g.Name, labels: tags,
				awsObject: awsObject{
					account: account, region: "us-east-1", arn: g.Arn,
					id: g.Id, service: "iam", objectType: "group",
				},
			}, tc))
	}
	return assets, nil
}

func cloudwatchLoggroups(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Name   string
	}
	loggroups, err := GetList[data](m, "return aws.cloudwatch.logGroups { arn name region }")
	if err != nil {
		return nil, err
	}
	for i := range loggroups {
		l := loggroups[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: l.Name, labels: make(map[string]string),
				awsObject: awsObject{
					account: account, region: l.Region, arn: l.Arn,
					id: l.Name, service: "cloudwatch", objectType: "loggroup",
				},
			}, tc))
	}
	return assets, nil
}

func lambdaFunctions(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Name   string
	}
	lambdafunctions, err := GetList[data](m, "return aws.lambda.functions { arn name region tags }")
	if err != nil {
		return nil, err
	}
	for i := range lambdafunctions {
		l := lambdafunctions[i]
		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: l.Name, labels: l.Tags,
				awsObject: awsObject{
					account: account, region: l.Region, arn: l.Arn,
					id: l.Name, service: "lambda", objectType: "function",
				},
			}, tc))
	}
	return assets, nil
}

func dynamodbTables(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Name   string
	}
	dynamodbtables, err := GetList[data](m, "return aws.dynamodb.tables { arn name region tags }")
	if err != nil {
		return nil, err
	}
	for i := range dynamodbtables {
		d := dynamodbtables[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: d.Name, labels: d.Tags,
				awsObject: awsObject{
					account: account, region: d.Region, arn: d.Arn,
					id: d.Name, service: "dynamodb", objectType: "table",
				},
			}, tc))
	}
	type gdata struct {
		Arn  string
		Tags map[string]string
		Name string
	}
	globaltables, err := GetList[gdata](m, "return aws.dynamodb.globalTables { arn name tags }")
	if err != nil {
		return nil, err
	}
	for i := range globaltables {
		g := globaltables[i]
		region := "us-east-1" // global service

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: g.Name, labels: g.Tags,
				awsObject: awsObject{
					account: account, region: region, arn: g.Arn,
					id: g.Name, service: "dynamodb", objectType: "table",
				},
			}, tc))
	}
	return assets, nil
}

func redshiftClusters(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Name   string
	}
	clusters, err := GetList[data](m, "return aws.redshift.clusters { arn name region tags }")
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		c := clusters[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: c.Name, labels: c.Tags,
				awsObject: awsObject{
					account: account, region: c.Region, arn: c.Arn,
					id: c.Name, service: "redshift", objectType: "cluster",
				},
			}, tc))
	}
	return assets, nil
}

func ec2Volumes(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Id     string
	}
	volumes, err := GetList[data](m, "return aws.ec2.volumes { arn id region tags }")
	if err != nil {
		return nil, err
	}
	for i := range volumes {
		v := volumes[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: v.Id, labels: v.Tags,
				awsObject: awsObject{
					account: account, region: v.Region, arn: v.Arn,
					id: v.Id, service: "ec2", objectType: "volume",
				},
			}, tc))
	}
	return assets, nil
}

func ec2Snapshots(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Id     string
	}
	snapshots, err := GetList[data](m, "return aws.ec2.snapshots { arn id region tags }")
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		s := snapshots[i]
		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: s.Id, labels: s.Tags,
				awsObject: awsObject{
					account: account, region: s.Region, arn: s.Arn,
					id: s.Id, service: "ec2", objectType: "snapshot",
				},
			}, tc))
	}
	return assets, nil
}

func ecsContainers(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn               string
		TaskDefinitionArn string
		Name              string
		PublicIp          string
		Image             string
		Region            string
		RuntimeId         string
		Status            string
		ClusterName       string
		ContainerName     string
	}
	containers, err := GetList[data](m, "return aws.ecs.containers { arn taskDefinitionArn name publicIp image region runtimeId status platformFamily platformVersion containerName }")
	if err != nil {
		return nil, err
	}
	for i := range containers {
		c := containers[i]
		tags := map[string]string{
			common.IPLabel:         c.PublicIp,
			ImageLabel:             c.Image,
			TaskDefinitionArnLabel: c.TaskDefinitionArn,
			RuntimeIdLabel:         c.RuntimeId,
			StateLabel:             c.Status,
			RegionLabel:            c.Region,
			ArnLabel:               c.Arn,
			ClusterNameLabel:       c.ClusterName,
			ContainerName:          c.ContainerName,
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: c.Name, labels: tags,
				awsObject: awsObject{
					account: account, region: c.Region, arn: c.Arn,
					id: c.Name, service: "ecs", objectType: "container",
				},
			}, tc))
	}
	return assets, nil
}

func ecsContainerInstances(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn              string
		AgentConnected   bool
		CapacityProvider string
		Id               string
		Region           string
		Ec2Instance      map[string]interface{}
	}

	instances, err := GetList[data](m, "return aws.ecs.containerInstances { arn region agentConnected capacityProvider id ec2Instance { arn instanceId tags region state image { name } } }")
	if err != nil {
		return nil, err
	}
	for i := range instances {
		inst := instances[i]
		name := inst.Id
		tags := map[string]string{AgentConnectedLabel: strconv.FormatBool(inst.AgentConnected), CapacityProviderLabel: inst.CapacityProvider}

		ec2Instance := inst.Ec2Instance

		if ec2Instance != nil {
			if ec2Instance["tags"] != nil {
				if val, ok := ec2Instance["tags"].(map[string]interface{})[AWSNameLabel]; ok {
					name = val.(string)
				}
			}
			if ec2Instance["state"] != nil {
				tags[StateLabel] = ec2Instance["state"].(string)
			}
			if ec2Instance["image"] != nil {
				if val, ok := ec2Instance["tags"].(map[string]interface{})[ImageNameLabel]; ok {
					tags[ImageNameLabel] = val.(string)
				}
			}
		}
		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: tags,
				awsObject: awsObject{
					account: account, region: inst.Region, arn: inst.Arn,
					id: inst.Id, service: "ecs", objectType: "instance",
				},
			}, tc))
	}

	return assets, nil
}

func ecrImages(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn      string
		Region   string
		Tags     []string
		Digest   string
		RepoName string
		Uri      string
	}
	images, err := GetList[data](m, "return aws.ecr.images { digest repoName arn tags region uri }")
	if err != nil {
		return nil, err
	}
	for i := range images {
		ecri := images[i]
		tags := make(map[string]string)
		for _, t := range ecri.Tags {
			tags["tag"] = t
		}
		tags[DigestLabel] = ecri.Digest
		tags[RepoUrlLabel] = ecri.Uri

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: aws.EcrImageName(aws.ImageInfo{RepoName: ecri.RepoName, Digest: ecri.Digest}), labels: tags,
				awsObject: awsObject{
					account: account, region: ecri.Region, arn: ecri.Arn,
					id: ecri.Digest, service: "ecr", objectType: "image",
				},
			}, tc))
	}
	return assets, nil
}

func ec2Instances(m *MqlDiscovery, account string, tc *providers.Config, whereFilter string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn        string
		Region     string
		Tags       map[string]string
		InstanceId string
		State      string
		Name       string
		Image      map[string]interface{}
		PublicIp   string
	}
	query := "return aws.ec2.instances { arn instanceId tags region state publicIp image { name id } }"
	if len(whereFilter) > 0 {
		query = fmt.Sprintf("return aws.ec2.instances.where(%s) { arn instanceId tags region state publicIp image { name id } }", whereFilter)
	}
	instances, err := GetList[data](m, query)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		inst := instances[i]
		name := inst.InstanceId
		if val, ok := inst.Tags[AWSNameLabel]; ok {
			name = val
		}
		inst.Tags[common.IPLabel] = inst.PublicIp
		inst.Tags[InstanceLabel] = inst.InstanceId
		inst.Tags[StateLabel] = inst.State
		if inst.Image != nil {
			inst.Tags[ImageNameLabel] = inst.Image["name"].(string)
			inst.Tags[ImageIdLabel] = inst.Image["id"].(string)
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: inst.Tags,
				awsObject: awsObject{
					account: account, region: inst.Region, arn: inst.Arn,
					id: inst.InstanceId, service: "ec2", objectType: "instance",
				},
			}, tc))
	}
	return assets, nil
}

func ssmInstances(m *MqlDiscovery, account string, tc *providers.Config, whereFilter string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn          string
		Region       string
		InstanceId   string
		PlatformName string
		PingStatus   string
		Tags         map[string]string
		IPAddress    string
	}
	query := "return aws.ssm.instances { arn instanceId pingStatus platformName region tags ipAddress }"
	if len(whereFilter) > 0 {
		query = fmt.Sprintf("return aws.ssm.instances.where(%s) { arn instanceId pingStatus platformName region tags ipAddress }", whereFilter)
	}
	instances, err := GetList[data](m, query)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		inst := instances[i]
		inst.Tags[SSMPingLabel] = inst.PingStatus
		inst.Tags[PlatformLabel] = inst.PlatformName
		inst.Tags[InstanceLabel] = inst.InstanceId
		inst.Tags[common.IPLabel] = inst.IPAddress
		name := inst.InstanceId
		if val, ok := inst.Tags["Name"]; ok {
			name = val
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: name, labels: inst.Tags,
				awsObject: awsObject{
					account: account, region: inst.Region, arn: inst.Arn,
					id: inst.InstanceId, service: "ssm", objectType: "instance",
				},
			}, tc))
	}
	return assets, nil
}

func efsFilesystems(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Id     string
		Name   string
	}
	filesystems, err := GetList[data](m, "return aws.efs.filesystems { arn name region tags id }")
	if err != nil {
		return nil, err
	}
	for i := range filesystems {
		f := filesystems[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: f.Name, labels: f.Tags,
				awsObject: awsObject{
					account: account, region: f.Region, arn: f.Arn,
					id: f.Id, service: "efs", objectType: "filesystem",
				},
			}, tc))
	}
	return assets, nil
}

func gatewayRestApis(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Id     string
		Name   string
	}
	restapis, err := GetList[data](m, "return aws.apigateway.restApis { arn name region tags id }")
	if err != nil {
		return nil, err
	}
	for i := range restapis {
		r := restapis[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: r.Name, labels: r.Tags,
				awsObject: awsObject{
					account: account, region: r.Region, arn: r.Arn,
					id: r.Id, service: "gateway", objectType: "restapi",
				},
			}, tc))
	}
	return assets, nil
}

func elbLoadBalancers(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn  string
		Name string
	}
	loadbalancers, err := GetList[data](m, "return aws.elb.loadBalancers { arn name }")
	if err != nil {
		return nil, err
	}
	for i := range loadbalancers {
		lb := loadbalancers[i]
		var region string
		if arn.IsARN(lb.Arn) {
			if p, err := arn.Parse(lb.Arn); err == nil {
				region = p.Region
			}
		}

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: lb.Name, labels: make(map[string]string),
				awsObject: awsObject{
					account: account, region: region, arn: lb.Arn,
					id: lb.Name, service: "elb", objectType: "loadbalancer",
				},
			}, tc))
	}
	return assets, nil
}

func esDomains(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Name   string
	}
	domains, err := GetList[data](m, "return aws.es.domains { arn name region tags }")
	if err != nil {
		return nil, err
	}
	for i := range domains {
		d := domains[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: d.Name, labels: d.Tags,
				awsObject: awsObject{
					account: account, region: d.Region, arn: d.Arn,
					id: d.Name, service: "es", objectType: "domain",
				},
			}, tc))
	}
	return assets, nil
}

func kmsKeys(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Id     string
	}
	keys, err := GetList[data](m, "return aws.kms.keys { arn region id }")
	if err != nil {
		return nil, err
	}
	for i := range keys {
		k := keys[i]
		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: k.Id, labels: make(map[string]string),
				awsObject: awsObject{
					account: account, region: k.Region, arn: k.Arn,
					id: k.Id, service: "kms", objectType: "key",
				},
			}, tc))
	}
	return assets, nil
}

func sagemakerNotebookInstances(m *MqlDiscovery, account string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type data struct {
		Arn    string
		Region string
		Tags   map[string]string
		Name   string
	}
	notebookinstances, err := GetList[data](m, "return aws.sagemaker.notebookInstances { arn name region tags }")
	if err != nil {
		return nil, err
	}
	for i := range notebookinstances {
		n := notebookinstances[i]

		assets = append(assets, MqlObjectToAsset(account,
			mqlObject{
				name: n.Name, labels: n.Tags,
				awsObject: awsObject{
					account: account, region: n.Region, arn: n.Arn,
					id: n.Name, service: "sagemaker", objectType: "notebookinstance",
				},
			}, tc))
	}
	return assets, nil
}
