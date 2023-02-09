package aws

import (
	"errors"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	awsprovider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/resources"
	resource_pack "go.mondoo.com/cnquery/resources/packs/aws"
)

type MqlDiscovery struct {
	rt *resources.Runtime
}

func NewMQLAssetsDiscovery(provider *awsprovider.Provider) (*MqlDiscovery, error) {
	m, err := motor.New(provider)
	if err != nil {
		return nil, err
	}
	rt := resources.NewRuntime(resource_pack.Registry, m)
	return &MqlDiscovery{rt: rt}, nil
}

func (md *MqlDiscovery) Close() {
	if md.rt != nil && md.rt.Motor != nil {
		md.rt.Motor.Close()
	}
}

func GetList[T any](md *MqlDiscovery, query string) ([]T, error) {
	mqlExecutor := mql.New(md.rt, cnquery.DefaultFeatures)
	value, err := mqlExecutor.Exec(query, map[string]*llx.Primitive{})
	if err != nil {
		return nil, err
	}
	if value.Error != nil {
		return nil, value.Error
	}
	var out []T
	if err := mapstructure.Decode(value.Value, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func GatherMQLObjects(provider *awsprovider.Provider, tc *providers.Config, account string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	m, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}

	// todo: when the dedup story is in we should turn these on with the others
	if tc.IncludesOneOfDiscoveryTarget(DiscoveryECSContainersAPI) {
		if a, err := ecsContainers(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ecs containers")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(DiscoveryECRImageAPI) {
		if a, err := ecrImages(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ecr images")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(DiscoveryEC2InstanceAPI) {
		if a, err := ec2Instances(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ec2 instances")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(DiscoverySSMInstanceAPI) {
		if a, err := ssmInstances(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ssm instances")
		}
	}
	// end todo

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryS3Buckets) {
		if a, err := s3Buckets(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query s3 buckets")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryCloudtrailTrails) {
		if a, err := cloudtrailTrails(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query cloudtrail trails")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryRdsDbInstances) {
		if a, err := rdsInstances(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query rds instances")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryVPCs) {
		if a, err := vpcs(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query vpcs")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoverySecurityGroups) {
		if a, err := securityGroups(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query security groups")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryIAMUsers) {
		if a, err := iamUsers(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query iam users")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryIAMGroups) {
		if a, err := iamGroups(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query iam groups")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryCloudwatchLoggroups) {
		if a, err := cloudwatchLoggroups(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query cloudwatch log groups")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryLambdaFunctions) {
		if a, err := lambdaFunctions(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query lambda functions")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryDynamoDBTables) {
		if a, err := dynamodbTables(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query dynamodb tables")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryRedshiftClusters) {
		if a, err := redshiftClusters(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query redshift clusters")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryVolumes) {
		if a, err := ec2Volumes(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ec2 volumes")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoverySnapshots) {
		if a, err := ec2Snapshots(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query ec2 snapshots")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryEFSFilesystems) {
		if a, err := efsFilesystems(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query efs filesystems")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryAPIGatewayRestAPIs) {
		if a, err := gatewayRestApis(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query gateway restapis")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryELBLoadBalancers) {
		if a, err := elbLoadBalancers(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query elb loadbalancers")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryESDomains) {
		if a, err := esDomains(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query es domains")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryKMSKeys) {
		if a, err := kmsKeys(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query kms keys")
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoverySagemakerNotebookInstances) {
		if a, err := sagemakerNotebookInstances(m, account, tc); err == nil {
			assets = append(assets, a...)
		} else {
			log.Error().Err(err).Msg("unable to query sagemaker notebook instances")
		}
	}

	return assets, nil
}

type mqlObject struct {
	name      string
	labels    map[string]string
	awsObject awsObject
}

type awsObject struct {
	account    string
	region     string
	id         string
	service    string
	objectType string
	arn        string
}

type awsObjectPlatformInfo struct {
	title    string
	platform string
}

func MondooObjectID(awsObject awsObject) string {
	return "//platformid.api.mondoo.app/runtime/aws/" + awsObject.service + "/v1/accounts/" + awsObject.account + "/regions/" + awsObject.region + "/" + awsObject.objectType + "/" + awsObject.id
}

func MqlObjectToAsset(account string, mqlObject mqlObject, tc *providers.Config) *asset.Asset {
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.awsObject.id
	}
	if err := validate(mqlObject); err != nil {
		log.Error().Err(err).Msg("missing values in mql object to asset translation")
		return nil
	}
	info, err := getTitleFamily(mqlObject.awsObject)
	if err != nil {
		log.Error().Err(err).Msg("missing runtime info")
		return nil
	}
	platformid := MondooObjectID(mqlObject.awsObject)
	t := tc.Clone()
	t.PlatformId = platformid
	return &asset.Asset{
		PlatformIds: []string{platformid, mqlObject.awsObject.arn},
		Name:        mqlObject.name,
		Platform: &platform.Platform{
			Name:    info.platform,
			Title:   info.title,
			Kind:    providers.Kind_KIND_AWS_OBJECT,
			Runtime: providers.RUNTIME_AWS,
		},
		State:       asset.State_STATE_ONLINE,
		Labels:      addInformationalLabels(mqlObject.labels, mqlObject),
		Connections: []*providers.Config{t},
	}
}

func validate(m mqlObject) error {
	if m.name == "" {
		return errors.New("name required for mql aws object to asset translation")
	}
	if m.awsObject.id == "" {
		return errors.New("id required for mql aws object to asset translation")
	}
	if m.awsObject.region == "" {
		return errors.New("region required for mql aws object to asset translation")
	}
	if m.awsObject.account == "" {
		return errors.New("account required for mql aws object to asset translation")
	}
	if m.awsObject.arn == "" {
		return errors.New("arn required for mql aws object to asset translation")
	}
	return nil
}

func addInformationalLabels(l map[string]string, o mqlObject) map[string]string {
	if l == nil {
		l = make(map[string]string)
	}
	l[RegionLabel] = o.awsObject.region
	l[common.ParentId] = o.awsObject.account
	l["arn"] = o.awsObject.arn

	return l
}
