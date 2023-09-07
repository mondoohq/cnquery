// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/utils/stringx"
)

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.AwsConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	res, err := NewResource(runtime, "aws.account", map[string]*llx.RawData{"id": llx.StringData("aws.account/" + conn.AccountId())})
	if err != nil {
		return nil, err
	}

	awsAccount := res.(*mqlAwsAccount)

	targets := handleTargets(conn.Conf.Discover.Targets)
	for i := range targets {
		target := targets[i]
		list, err := discover(runtime, awsAccount, target)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}

	return in, nil
}

func handleTargets(targets []string) []string {
	if len(targets) == 0 {
		// default to auto if none defined
		return []string{connection.DiscoveryAccounts}
	}
	if stringx.Contains(targets, connection.DiscoveryAll) {
		return connection.All
	}
	if stringx.Contains(targets, connection.DiscoveryAuto) {
		return connection.Auto
	}
	if stringx.Contains(targets, connection.DiscoveryResources) {
		targets = remove(targets, connection.DiscoveryResources)
		targets = append(targets, connection.AllAPIResources...)
	}
	return targets
}

func discover(runtime *plugin.Runtime, awsAccount *mqlAwsAccount, target string) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	accountId := awsAccount.Id.Data
	assetList := []*inventory.Asset{}
	switch target {
	case connection.DiscoveryAccounts:
		assetList = append(assetList, accountAsset(conn, awsAccount))

	case connection.DiscoveryInstances:
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
			assetList = append(assetList, addConnectionInfoToEc2Asset(instance, accountId, conn))
		}
	case connection.DiscoverySSMInstances:
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
			if instance.GetSsm() != nil {
				if s := instance.GetSsm().Data.(map[string]interface{})["PingStatus"]; s != nil && s == "Online" {
					assetList = append(assetList, addSSMConnectionInfoToEc2Asset(instance, accountId, conn.Profile()))
				}
			}
		}
		res, err = NewResource(runtime, "aws.ssm", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}

		ssm := res.(*mqlAwsSsm)

		ins = ssm.GetInstances()
		if ins == nil {
			return assetList, nil
		}

		for i := range ins.Data {
			instance := ins.Data[i].(*mqlAwsSsmInstance)
			assetList = append(assetList, addConnectionInfoToSSMAsset(instance, accountId, conn))
		}
	case connection.DiscoveryECR:
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
			assetList = append(assetList, addConnectionInfoToEcrAsset(a, conn.Profile()))
		}
	case connection.DiscoveryECS:
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
			assetList = append(assetList, addConnectionInfoToECSContainerAsset(c, accountId, conn))
		}
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
	// case connection.DiscoveryECSContainersAPI:
	// case connection.DiscoveryECRImageAPI:
	// case connection.DiscoveryEC2InstanceAPI:
	// case connection.DiscoverySSMInstanceAPI:
	case connection.DiscoveryS3Buckets:
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
	case connection.DiscoveryCloudtrailTrails:
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
	case connection.DiscoveryRdsDbInstances:
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
	case connection.DiscoveryVPCs:
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
	case connection.DiscoverySecurityGroups:
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
	case connection.DiscoveryIAMGroups:
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
	case connection.DiscoveryCloudwatchLoggroups:
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
	case connection.DiscoveryLambdaFunctions:
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
	case connection.DiscoveryDynamoDBTables:
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
		ts = d.GetGlobalTables()
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
	case connection.DiscoveryIAMUsers:
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
	case connection.DiscoveryRedshiftClusters:
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
	case connection.DiscoveryVolumes:
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
	case connection.DiscoverySnapshots:
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
	case connection.DiscoveryEFSFilesystems:
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
	case connection.DiscoveryAPIGatewayRestAPIs:
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
	case connection.DiscoveryELBLoadBalancers:
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
	case connection.DiscoveryESDomains:
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
	case connection.DiscoveryKMSKeys:
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
	case connection.DiscoverySagemakerNotebookInstances:
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
