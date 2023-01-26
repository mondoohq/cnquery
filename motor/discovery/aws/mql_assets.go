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

func (md *MqlDiscovery) GetList(query string) []interface{} {
	mqlExecutor := mql.New(md.rt, cnquery.DefaultFeatures)
	value, err := mqlExecutor.Exec(query, map[string]*llx.Primitive{})
	if err != nil {
		return nil
	}

	a := []interface{}{}
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &a,
	})
	d.Decode(value.Value)
	return a
}

func GatherMQLObjects(provider *awsprovider.Provider, tc *providers.Config, account string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	m, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryS3Buckets) {
		assets = append(assets, s3Buckets(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryCloudtrailTrails) {
		assets = append(assets, cloudtrailTrails(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryRdsDbInstances) {
		assets = append(assets, rdsInstances(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryVPCs) {
		assets = append(assets, vpcs(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoverySecurityGroups) {
		assets = append(assets, securityGroups(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryIAMUsers) {
		assets = append(assets, iamUsers(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryIAMGroups) {
		assets = append(assets, iamGroups(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryCloudwatchLoggroups) {
		assets = append(assets, cloudwatchLoggroups(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryLambdaFunctions) {
		assets = append(assets, lambdaFunctions(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryDynamoDBTables) {
		assets = append(assets, dynamodbTables(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryRedshiftClusters) {
		assets = append(assets, redshiftClusters(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryVolumes) {
		assets = append(assets, ec2Volumes(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoverySnapshots) {
		assets = append(assets, ec2Snapshots(m, account, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryResources, DiscoveryECSContainersAPI) {
		assets = append(assets, ecsContainers(m, account, tc)...)
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
