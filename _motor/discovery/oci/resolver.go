package oci

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	DiscoveryAccounts = "accounts"
)

var ResourceDiscoveryTargets = []string{}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "OCI Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	discovery := []string{
		common.DiscoveryAuto, common.DiscoveryAll,
	}
	return discovery
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	m, err := resolver.NewMotorConnection(ctx, tc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	provider, ok := m.Provider.(*oci.Provider)
	if !ok {
		return nil, errors.New("could not create oci transport")
	}
	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(provider)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// add asset for the api itself
	info, err := provider.Tenant(ctx)
	if err != nil {
		return nil, err
	}

	var resolvedRoot *asset.Asset
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryAccounts) {
		name := root.Name
		if name == "" {
			name = AssembleIntegrationName(*info.Name)
		}

		resolvedRoot = &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        name,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			State:       asset.State_STATE_ONLINE,
		}
		resolved = append(resolved, resolvedRoot)
	}

	//// filter assets
	//discoverFilter := map[string]string{}
	//if tc.Discover != nil {
	//	discoverFilter = tc.Discover.Filter
	//}
	//
	//mqldiscovery, err := NewMQLAssetsDiscovery(provider)
	//if err != nil {
	//	return nil, err
	//}

	assetMap := make(map[string]*asset.Asset)
	// ensure we don't return the same asset twice
	for i := range resolved {
		assetMap[resolved[i].PlatformIds[0]] = resolved[i]
	}
	new := make([]*asset.Asset, 0, len(assetMap))
	for _, v := range assetMap {
		new = append(new, v)
	}

	return new, nil
}

func AssembleIntegrationName(id string) string {
	return fmt.Sprintf("OCI Tenant %s", id)
}
