package gcp

import (
	"context"
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
	gcpprovider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/resources"
	resource_pack "go.mondoo.com/cnquery/resources/packs/gcp"
)

type MqlDiscovery struct {
	rt *resources.Runtime
}

func NewMQLAssetsDiscovery(provider *gcpprovider.Provider) (*MqlDiscovery, error) {
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

func GatherAssets(ctx context.Context, tc *providers.Config, project string, credsResolver vault.Resolver, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	// Note: we use the resolver instead of the direct gcp_provider.New to resolve credentials properly
	pCfg := tc.Clone()
	motor, err := resolver.NewMotorConnection(ctx, pCfg, credsResolver)
	if err != nil {
		return nil, err
	}
	defer motor.Close()

	provider, ok := motor.Provider.(*gcpprovider.Provider)
	if !ok {
		return nil, errors.New("could not create gcp provider")
	}
	m, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		assets = append(assets, computeInstances(m, project, tc, sfn)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryComputeImages) {
		assets = append(assets, computeImages(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryComputeNetworks) {
		assets = append(assets, computeNetworks(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryComputeSubnetworks) {
		assets = append(assets, computeSubnetworks(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryComputeFirewalls) {
		assets = append(assets, computeFirewalls(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryGkeClusters) {
		assets = append(assets, gkeClusters(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryStorageBuckets) {
		assets = append(assets, storageBuckets(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryBigQueryDatasets) {
		assets = append(assets, bigQueryDatasets(m, project, tc)...)
	}

	return assets, nil
}

type mqlObject struct {
	name      string
	labels    map[string]string
	gcpObject gcpObject
}

type gcpObject struct {
	project    string
	region     string
	id         string
	service    string
	objectType string
	name       string
}

type gcpObjectPlatformInfo struct {
	title    string
	platform string
}

func GcpPlatformID(o gcpObject) string {
	return "//platformid.api.mondoo.app/runtime/gcp/" + o.service + "/v1/projects/" + o.project + "/regions/" + o.region + "/" + o.objectType + "/" + o.name
}

func MqlObjectToAsset(account string, mqlObject mqlObject, tc *providers.Config) *asset.Asset {
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.gcpObject.id
	}
	if err := validate(mqlObject); err != nil {
		log.Error().Err(err).Msg("missing values in mql object to asset translation")
		return nil
	}
	info, err := getTitleFamily(mqlObject.gcpObject)
	if err != nil {
		log.Error().Err(err).Msg("missing runtime info")
		return nil
	}
	platformid := GcpPlatformID(mqlObject.gcpObject)
	t := tc.Clone()
	t.PlatformId = platformid
	return &asset.Asset{
		PlatformIds: []string{platformid},
		Name:        mqlObject.name,
		Platform: &platform.Platform{
			Name:    info.platform,
			Title:   info.title,
			Kind:    providers.Kind_KIND_GCP_OBJECT,
			Runtime: providers.RUNTIME_GCP,
		},
		State:       asset.State_STATE_ONLINE,
		Labels:      addInformationalLabels(mqlObject.labels, mqlObject),
		Connections: []*providers.Config{t},
	}
}

func validate(m mqlObject) error {
	if m.name == "" {
		return errors.New("name required for mql gcp object to asset translation")
	}
	if m.gcpObject.id == "" {
		return errors.New("id required for mql gcp object to asset translation")
	}
	if m.gcpObject.region == "" {
		return errors.New("region required for mql gcp object to asset translation")
	}
	if m.gcpObject.project == "" {
		return errors.New("project required for mql gcp object to asset translation")
	}
	if m.gcpObject.name == "" {
		return errors.New("name required for mql gcp object to asset translation")
	}
	return nil
}

func addInformationalLabels(l map[string]string, o mqlObject) map[string]string {
	if l == nil {
		l = make(map[string]string)
	}
	l[RegionLabel] = o.gcpObject.region
	l[common.ParentId] = o.gcpObject.project
	l[ProjectLabel] = o.gcpObject.project
	return l
}
