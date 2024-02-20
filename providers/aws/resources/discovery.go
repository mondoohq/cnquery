// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

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
	DiscoveryRdsDbClusters              = "rds-dbclusters"
	DiscoveryVPCs                       = "vpcs"
	DiscoverySecurityGroups             = "security-groups"
	DiscoveryIAMUsers                   = "iam-users"
	DiscoveryIAMGroups                  = "iam-groups"
	DiscoveryCloudwatchLoggroups        = "cloudwatch-loggroups"
	DiscoveryLambdaFunctions            = "lambda-functions"
	DiscoveryDynamoDBTables             = "dynamodb-tables"
	DiscoveryDynamoDBGlobalTables       = "dynamodb-global-tables"
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
	DiscoveryRdsDbClusters,
	DiscoveryVPCs,
	DiscoverySecurityGroups,
	DiscoveryIAMUsers,
	DiscoveryIAMGroups,
	DiscoveryCloudwatchLoggroups,
	DiscoveryLambdaFunctions,
	DiscoveryDynamoDBTables,
	DiscoveryDynamoDBGlobalTables,
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

func contains(sl []string, s string) bool {
	for i := range sl {
		if sl[i] == s {
			return true
		}
	}
	return false
}

func containsInterfaceSlice(sl []interface{}, s string) bool {
	for i := range sl {
		if sl[i].(string) == s {
			return true
		}
	}
	return false
}

func instanceMatchesFilters(instance *mqlAwsEc2Instance, filters connection.DiscoveryFilters) bool {
	matches := true
	f := filters.Ec2DiscoveryFilters
	if len(f.Regions) > 0 {
		if !contains(f.Regions, instance.Region.Data) {
			matches = false
		}
	}
	if len(f.InstanceIds) > 0 {
		if !contains(f.InstanceIds, instance.InstanceId.Data) {
			matches = false
		}
	}
	if len(f.Tags) > 0 {
		for k, v := range f.Tags {
			if instance.Tags.Data[k] == nil {
				return false
			}
			if instance.Tags.Data[k].(string) != v {
				return false
			}
		}
	}
	return matches
}

func imageMatchesFilters(image *mqlAwsEcrImage, filters connection.DiscoveryFilters) bool {
	f := filters.EcrDiscoveryFilters
	if len(f.Tags) > 0 {
		for i := range f.Tags {
			t := f.Tags[i]
			if !containsInterfaceSlice(image.Tags.Data, t) {
				return false
			}
		}
	}
	return true
}

func containerMatchesFilters(container *mqlAwsEcsContainer, filters connection.DiscoveryFilters) bool {
	f := filters.EcsDiscoveryFilters
	if f.OnlyRunningContainers {
		if container.Status.Data != "RUNNING" {
			return false
		}
	}
	return true
}

func shouldScanEcsContainerInstances(filters connection.DiscoveryFilters) bool {
	return filters.EcsDiscoveryFilters.DiscoverInstances
}

func shouldScanEcsContainerImages(filters connection.DiscoveryFilters) bool {
	return filters.EcsDiscoveryFilters.DiscoverImages
}

func discoveredAssetMatchesGeneralFilters(asset *inventory.Asset, filters connection.GeneralResourceDiscoveryFilters) bool {
	if len(filters.Tags) > 0 {
		for k, v := range filters.Tags {
			if asset.Labels[k] != v {
				return false
			}
		}
	}
	return true
}

func Discover(runtime *plugin.Runtime, filters connection.DiscoveryFilters) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.AwsConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	res, err := NewResource(runtime, "aws.account", map[string]*llx.RawData{"id": llx.StringData("aws.account/" + conn.AccountId())})
	if err != nil {
		return nil, err
	}
	var awsAccount *mqlAwsAccount
	if res != nil {
		awsAccount = res.(*mqlAwsAccount)
	}

	targets := handleTargets(conn.Conf.Discover.Targets)
	for i := range targets {
		target := targets[i]
		list, err := discover(runtime, awsAccount, target, filters)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		if len(filters.GeneralDiscoveryFilters.Tags) > 0 {
			newList := []*inventory.Asset{}
			for i := range list {
				if discoveredAssetMatchesGeneralFilters(list[i], filters.GeneralDiscoveryFilters) {
					newList = append(newList, list[i])
				}
			}
			list = newList
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}
	return in, nil
}

func handleTargets(targets []string) []string {
	if len(targets) == 0 || stringx.Contains(targets, DiscoveryAuto) {
		// default to auto if none defined
		return Auto
	}

	if stringx.Contains(targets, DiscoveryAll) {
		return All
	}
	if stringx.Contains(targets, DiscoveryResources) {
		targets = remove(targets, DiscoveryResources)
		targets = append(targets, AllAPIResources...)
	}
	return targets
}

// for now we have to post process the filters
// more ideally, we should pass the filters in when discovering
// so that we dont unnecessarily discover assets we will later discard
func discover(runtime *plugin.Runtime, awsAccount *mqlAwsAccount, target string, filters connection.DiscoveryFilters) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	accountId := trimAwsAccountIdToJustId(awsAccount.Id.Data)
	assetList := []*inventory.Asset{}
	switch target {
	case DiscoveryAccounts:
		assetList = append(assetList, accountAsset(conn, awsAccount))

	case DiscoveryInstances:
		res, err := NewResource(runtime, "aws.ec2", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ec2 := res.(*mqlAwsEc2)

		ins := ec2.GetInstances()
		if ins == nil {
			return assetList, nil
		}

		for i := range ins.Data {
			instance := ins.Data[i].(*mqlAwsEc2Instance)
			if !instanceMatchesFilters(instance, filters) {
				continue
			}
			assetList = append(assetList, addConnectionInfoToEc2Asset(instance, accountId, conn))

		}
	case DiscoverySSMInstances:
		res, err := NewResource(runtime, "aws.ssm", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ssm := res.(*mqlAwsSsm)

		ins := ssm.GetInstances()
		if ins == nil {
			return assetList, nil
		}

		for i := range ins.Data {
			instance := ins.Data[i].(*mqlAwsSsmInstance)
			assetList = append(assetList, addConnectionInfoToSSMAsset(instance, accountId, conn))
		}
	case DiscoveryECR:
		res, err := NewResource(runtime, "aws.ecr", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ecr := res.(*mqlAwsEcr)

		images := ecr.GetImages()
		if images == nil {
			return assetList, nil
		}

		for i := range images.Data {
			a := images.Data[i].(*mqlAwsEcrImage)
			if !imageMatchesFilters(a, filters) {
				continue
			}
			ecrAsset := addConnectionInfoToEcrAsset(a, conn)
			if len(ecrAsset.Connections) > 0 {
				assetList = append(assetList, ecrAsset)
			} else {
				log.Warn().Str("name", ecrAsset.Name).Msg("cannot scan ecr image with no tag")
			}
		}
	case DiscoveryECS:
		res, err := NewResource(runtime, "aws.ecs", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ecs := res.(*mqlAwsEcs)

		containers := ecs.GetContainers()
		if containers == nil {
			return assetList, nil
		}

		for i := range containers.Data {
			c := containers.Data[i].(*mqlAwsEcsContainer)
			if !containerMatchesFilters(c, filters) {
				continue
			}
			assetList = append(assetList, addConnectionInfoToECSContainerAsset(c, accountId, conn))
		}
		if shouldScanEcsContainerInstances(filters) {
			containerInst := ecs.GetContainerInstances()
			if containerInst == nil {
				return assetList, nil
			}

			for i := range containerInst.Data {
				if a, ok := containerInst.Data[i].(*mqlAwsEc2Instance); ok {
					assetList = append(assetList, addConnectionInfoToEc2Asset(a, accountId, conn))
				} else if b, ok := containerInst.Data[i].(*mqlAwsEcsInstance); ok {
					assetList = append(assetList, addConnectionInfoToECSContainerInstanceAsset(b, accountId, conn))
				}
			}
		}
	case DiscoveryECSContainersAPI:
		res, err := NewResource(runtime, "aws.ecs", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ecs := res.(*mqlAwsEcs)

		containers := ecs.GetContainers()
		if containers == nil {
			return assetList, nil
		}

		for i := range containers.Data {
			c := containers.Data[i].(*mqlAwsEcsContainer)
			if !containerMatchesFilters(c, filters) {
				continue
			}
			assetList = append(assetList, MqlObjectToAsset(accountId,
				mqlObject{
					name: c.ContainerName.Data, labels: map[string]string{},
					awsObject: awsObject{
						account: accountId, region: c.Region.Data, arn: c.Arn.Data,
						id: c.Arn.Data, service: "ecs", objectType: "container",
					},
				}, conn))
		}

	case DiscoveryECRImageAPI:
		res, err := NewResource(runtime, "aws.ecr", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ecr := res.(*mqlAwsEcr)

		images := ecr.GetImages()
		if images == nil {
			return assetList, nil
		}

		for i := range images.Data {
			a := images.Data[i].(*mqlAwsEcrImage)
			if !imageMatchesFilters(a, filters) {
				continue
			}
			l := make(map[string]string)
			for i := range a.Tags.Data {
				l[a.Tags.Data[i].(string)] = ""
			}

			assetList = append(assetList, MqlObjectToAsset(accountId,
				mqlObject{
					name: l["Name"], labels: l,
					awsObject: awsObject{
						account: accountId, region: a.Region.Data, arn: a.Arn.Data,
						id: a.Uri.Data, service: "ecr", objectType: "image",
					},
				}, conn))
		}
	case DiscoveryEC2InstanceAPI:
		res, err := NewResource(runtime, "aws.ec2", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ec2 := res.(*mqlAwsEc2)

		ins := ec2.GetInstances()
		if ins == nil {
			return assetList, nil
		}

		for i := range ins.Data {
			instance := ins.Data[i].(*mqlAwsEc2Instance)
			if !instanceMatchesFilters(instance, filters) {
				continue
			}
			l := mapStringInterfaceToStringString(instance.Tags.Data)
			assetList = append(assetList, MqlObjectToAsset(accountId,
				mqlObject{
					name: getInstanceName(instance.InstanceId.Data, l), labels: l,
					awsObject: awsObject{
						account: accountId, region: instance.Region.Data, arn: instance.Arn.Data,
						id: instance.InstanceId.Data, service: "ec2", objectType: "instance",
					},
				}, conn))
		}
	case DiscoverySSMInstanceAPI:
		res, err := NewResource(runtime, "aws.ssm", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ssm := res.(*mqlAwsSsm)

		ins := ssm.GetInstances()
		if ins == nil {
			return assetList, nil
		}

		for i := range ins.Data {
			instance := ins.Data[i].(*mqlAwsSsmInstance)
			l := mapStringInterfaceToStringString(instance.Tags.Data)
			assetList = append(assetList, MqlObjectToAsset(accountId,
				mqlObject{
					name: getInstanceName(instance.InstanceId.Data, l), labels: l,
					awsObject: awsObject{
						account: accountId, region: instance.Region.Data, arn: instance.Arn.Data,
						id: instance.InstanceId.Data, service: "ssm", objectType: "instance",
					},
				}, conn))
		}
	case DiscoveryS3Buckets:
		res, err := NewResource(runtime, "aws.s3", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		s := res.(*mqlAwsS3)

		bs := s.GetBuckets()
		if bs == nil {
			return assetList, nil
		}

		for i := range bs.Data {
			f := bs.Data[i].(*mqlAwsS3Bucket)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Location.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "s3", objectType: "bucket",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryCloudtrailTrails:
		res, err := NewResource(runtime, "aws.cloudtrail", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		l := res.(*mqlAwsCloudtrail)

		fs := l.GetTrails()
		if fs == nil {
			return assetList, nil
		}

		for i := range fs.Data {
			f := fs.Data[i].(*mqlAwsCloudtrailTrail)

			m := mqlObject{
				name: f.Name.Data, labels: map[string]string{},
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "cloudtrail", objectType: "trail",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryRdsDbInstances:
		res, err := NewResource(runtime, "aws.rds", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		r := res.(*mqlAwsRds)

		dbs := r.GetDbInstances()
		if dbs == nil {
			return assetList, nil
		}

		for i := range dbs.Data {
			f := dbs.Data[i].(*mqlAwsRdsDbinstance)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "rds", objectType: "dbinstance",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryRdsDbClusters:
		res, err := NewResource(runtime, "aws.rds", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		r := res.(*mqlAwsRds)

		clusters := r.GetDbClusters()
		if clusters == nil {
			return assetList, nil
		}

		for i := range clusters.Data {
			f := clusters.Data[i].(*mqlAwsRdsDbcluster)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Id.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "rds", objectType: "dbcluster",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryVPCs:
		res, err := NewResource(runtime, "aws", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		a := res.(*mqlAws)

		vpcs := a.GetVpcs()
		if vpcs == nil {
			return assetList, nil
		}

		for i := range vpcs.Data {
			f := vpcs.Data[i].(*mqlAwsVpc)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Id.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "vpc", objectType: "vpc",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoverySecurityGroups:
		res, err := NewResource(runtime, "aws.ec2", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsEc2)

		sgs := e.GetSecurityGroups()
		if sgs == nil {
			return assetList, nil
		}

		for i := range sgs.Data {
			f := sgs.Data[i].(*mqlAwsEc2Securitygroup)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "ec2", objectType: "securitygroup",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryIAMGroups:
		res, err := NewResource(runtime, "aws.iam", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		iam := res.(*mqlAwsIam)

		groups := iam.GetGroups()
		if groups == nil {
			return assetList, nil
		}

		for i := range groups.Data {
			group := groups.Data[i].(*mqlAwsIamGroup)
			labels := map[string]string{}

			m := mqlObject{
				name: group.Name.Data, labels: labels,
				awsObject: awsObject{
					account: accountId, region: "us-east-1", arn: group.Arn.Data,
					id: group.Id.Data, service: "iam", objectType: "group",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryCloudwatchLoggroups:
		res, err := NewResource(runtime, "aws.cloudwatch", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		cw := res.(*mqlAwsCloudwatch)

		lgroups := cw.GetLogGroups()
		if lgroups == nil {
			return assetList, nil
		}

		for i := range lgroups.Data {
			group := lgroups.Data[i].(*mqlAwsCloudwatchLoggroup)
			labels := map[string]string{}

			m := mqlObject{
				name: group.Name.Data, labels: labels,
				awsObject: awsObject{
					account: accountId, region: group.Region.Data, arn: group.Arn.Data,
					id: group.Name.Data, service: "cloudwatch", objectType: "loggroup",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryLambdaFunctions:
		res, err := NewResource(runtime, "aws.lambda", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		l := res.(*mqlAwsLambda)

		fs := l.GetFunctions()
		if fs == nil {
			return assetList, nil
		}

		for i := range fs.Data {
			f := fs.Data[i].(*mqlAwsLambdaFunction)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "lambda", objectType: "function",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryDynamoDBTables:
		res, err := NewResource(runtime, "aws.dynamodb", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		d := res.(*mqlAwsDynamodb)

		ts := d.GetTables()
		if ts == nil {
			return assetList, nil
		}

		for i := range ts.Data {
			f := ts.Data[i].(*mqlAwsDynamodbTable)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "dynamodb", objectType: "table",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryDynamoDBGlobalTables:
		res, err := NewResource(runtime, "aws.dynamodb", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		d := res.(*mqlAwsDynamodb)

		ts := d.GetGlobalTables()
		if ts == nil {
			return assetList, nil
		}

		for i := range ts.Data {
			f := ts.Data[i].(*mqlAwsDynamodbGlobaltable)

			m := mqlObject{
				name: f.Name.Data, labels: map[string]string{},
				awsObject: awsObject{
					account: accountId, region: "us-east-1", arn: f.Arn.Data,
					id: f.Name.Data, service: "dynamodb", objectType: "table",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryIAMUsers:
		res, err := NewResource(runtime, "aws.iam", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		iam := res.(*mqlAwsIam)

		users := iam.GetUsers()
		if users == nil {
			return assetList, nil
		}

		for i := range users.Data {
			user := users.Data[i].(*mqlAwsIamUser)
			labels := map[string]string{}

			m := mqlObject{
				name: user.Name.Data, labels: labels,
				awsObject: awsObject{
					account: accountId, region: "us-east-1", arn: user.Arn.Data,
					id: user.Id.Data, service: "iam", objectType: "user",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryRedshiftClusters:
		res, err := NewResource(runtime, "aws.redshift", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		r := res.(*mqlAwsRedshift)

		cs := r.GetClusters()
		if cs == nil {
			return assetList, nil
		}

		for i := range cs.Data {
			f := cs.Data[i].(*mqlAwsRedshiftCluster)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "redshift", objectType: "cluster",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryVolumes:
		res, err := NewResource(runtime, "aws.ec2", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsEc2)

		vs := e.GetVolumes()
		if vs == nil {
			return assetList, nil
		}

		for i := range vs.Data {
			f := vs.Data[i].(*mqlAwsEc2Volume)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Id.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "ec2", objectType: "volume",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoverySnapshots:
		res, err := NewResource(runtime, "aws.ec2", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsEc2)

		s := e.GetSnapshots()
		if s == nil {
			return assetList, nil
		}

		for i := range s.Data {
			f := s.Data[i].(*mqlAwsEc2Snapshot)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Id.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "ec2", objectType: "snapshot",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryEFSFilesystems:
		res, err := NewResource(runtime, "aws.efs", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsEfs)

		fs := e.GetFilesystems()
		if fs == nil {
			return assetList, nil
		}

		for i := range fs.Data {
			f := fs.Data[i].(*mqlAwsEfsFilesystem)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "efs", objectType: "filesystem",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryAPIGatewayRestAPIs:
		res, err := NewResource(runtime, "aws.apigateway", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsApigateway)

		ras := e.GetRestApis()
		if ras == nil {
			return assetList, nil
		}

		for i := range ras.Data {
			f := ras.Data[i].(*mqlAwsApigatewayRestapi)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "gateway", objectType: "restapi",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryELBLoadBalancers:
		res, err := NewResource(runtime, "aws.elb", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsElb)

		lbs := e.GetLoadBalancers()
		if lbs == nil {
			return assetList, nil
		}

		for i := range lbs.Data {
			f := lbs.Data[i].(*mqlAwsElbLoadbalancer)
			var region string
			if arn.IsARN(f.Arn.Data) {
				if p, err := arn.Parse(f.Arn.Data); err == nil {
					region = p.Region
				}
			}
			m := mqlObject{
				name: f.Name.Data, labels: map[string]string{},
				awsObject: awsObject{
					account: accountId, region: region, arn: f.Arn.Data,
					id: f.Name.Data, service: "elb", objectType: "loadbalancer",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryESDomains:
		res, err := NewResource(runtime, "aws.es", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsEs)

		ras := e.GetDomains()
		if ras == nil {
			return assetList, nil
		}

		for i := range ras.Data {
			f := ras.Data[i].(*mqlAwsEsDomain)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "es", objectType: "domain",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoveryKMSKeys:
		res, err := NewResource(runtime, "aws.kms", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsKms)

		ras := e.GetKeys()
		if ras == nil {
			return assetList, nil
		}

		for i := range ras.Data {
			f := ras.Data[i].(*mqlAwsKmsKey)

			m := mqlObject{
				name: f.Id.Data, labels: map[string]string{},
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Id.Data, service: "kms", objectType: "key",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	case DiscoverySagemakerNotebookInstances:
		res, err := NewResource(runtime, "aws.sagemaker", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		e := res.(*mqlAwsSagemaker)

		ras := e.GetNotebookInstances()
		if ras == nil {
			return assetList, nil
		}

		for i := range ras.Data {
			f := ras.Data[i].(*mqlAwsSagemakerNotebookinstance)

			tags := mapStringInterfaceToStringString(f.Tags.Data)
			m := mqlObject{
				name: f.Name.Data, labels: tags,
				awsObject: awsObject{
					account: accountId, region: f.Region.Data, arn: f.Arn.Data,
					id: f.Name.Data, service: "sagemaker", objectType: "notebookinstance",
				},
			}
			assetList = append(assetList, MqlObjectToAsset(accountId, m, conn))
		}
	}
	return assetList, nil
}
