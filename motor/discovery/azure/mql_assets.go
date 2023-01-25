package azure

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
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/resources"

	azure "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	resource_pack "go.mondoo.com/cnquery/resources/packs/azure"
)

type MqlDiscovery struct {
	rt *resources.Runtime
}

func NewMQLAssetsDiscovery(provider *azure.Provider) (*MqlDiscovery, error) {
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

func GatherAssets(ctx context.Context, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	pCfg := tc.Clone()
	motor, err := resolver.NewMotorConnection(ctx, pCfg, credsResolver)
	if err != nil {
		return nil, err
	}
	defer motor.Close()

	provider, ok := motor.Provider.(*azure.Provider)
	if !ok {
		return nil, errors.New("could not create azure provider")
	}
	m, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		instances, err := computeInstances(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, instances...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoverySqlServers) {
		servers, err := computeSqlServers(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, servers...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryPostgresServers) {
		servers, err := computePostgresqlServers(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, servers...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryMySqlServers) {
		servers, err := computeMySqlServers(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, servers...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryMariaDbServers) {
		servers, err := computeMariaDbServers(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, servers...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryStorageAccounts) {
		accounts, err := computeStorageAccounts(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, accounts...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryStorageContainers) {
		containers, err := computeStorageAccountContainers(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, containers...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryKeyVaults) {
		vaults, err := computeKeyVaultsVaults(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, vaults...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoverySecurityGroups) {
		securityGroups, err := computeNetworkSecurityGroups(m, provider.SubscriptionID(), tc)
		if err != nil {
			return nil, err
		}
		assets = append(assets, securityGroups...)
	}

	return assets, nil
}

type mqlObject struct {
	name        string
	labels      map[string]string
	azureObject azureObject
}

type azureObject struct {
	subscription string
	id           string
	region       string
	service      string
	objectType   string
}

type azureObjectPlatformInfo struct {
	title    string
	platform string
}

func AzureObjectPlatformId(azureObject azureObject) string {
	// the azure resources have an unique id throughout the whole subscription
	// that should be enough for an unique platform id
	return "//platformid.api.mondoo.app/runtime/azure/v1" + azureObject.id
}

func MqlObjectToAsset(mqlObject mqlObject, tc *providers.Config) *asset.Asset {
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.azureObject.id
	}
	if err := validate(mqlObject); err != nil {
		log.Error().Err(err).Msg("missing values in mql object to asset translation")
		return nil
	}
	info, err := getTitleFamily(mqlObject.azureObject)
	if err != nil {
		log.Error().Err(err).Msg("missing runtime info")
		return nil
	}
	platformid := AzureObjectPlatformId(mqlObject.azureObject)
	t := tc.Clone()
	t.PlatformId = platformid
	return &asset.Asset{
		PlatformIds: []string{platformid, mqlObject.azureObject.id},
		Name:        mqlObject.name,
		Platform: &platform.Platform{
			Name:    info.platform,
			Title:   info.title,
			Kind:    providers.Kind_KIND_AZURE_OBJECT,
			Runtime: providers.RUNTIME_AZ,
		},
		State:       asset.State_STATE_ONLINE,
		Labels:      addInformationalLabels(mqlObject.labels, mqlObject),
		Connections: []*providers.Config{t},
	}
}

func validate(m mqlObject) error {
	if m.name == "" {
		return errors.New("name required for mql object to asset translation")
	}
	if m.azureObject.id == "" {
		return errors.New("id required for mql aws object to asset translation")
	}
	if m.azureObject.subscription == "" {
		return errors.New("sub required for mql aws object to asset translation")
	}
	if m.azureObject.region == "" {
		return errors.New("region required for mql aws object to asset translation")
	}

	return nil
}

func addInformationalLabels(l map[string]string, o mqlObject) map[string]string {
	if l == nil {
		l = make(map[string]string)
	}
	l[RegionLabel] = o.azureObject.region
	l[common.ParentId] = o.azureObject.subscription
	l[SubscriptionLabel] = o.azureObject.subscription
	return l
}
