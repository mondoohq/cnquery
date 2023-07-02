package gcp

import (
	"context"

	"errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

var ProjectDiscoveryTargets = []string{
	DiscoveryInstances,
	DiscoveryComputeImages,
	DiscoveryComputeNetworks,
	DiscoveryComputeSubnetworks,
	DiscoveryComputeFirewalls,
	DiscoveryGkeClusters,
	DiscoveryStorageBuckets,
	DiscoveryBigQueryDatasets,
}

type GcpProjectResolver struct{}

func (k *GcpProjectResolver) Name() string {
	return "GCP Project Resolver"
}

func (r *GcpProjectResolver) AvailableDiscoveryTargets() []string {
	return append(ProjectDiscoveryTargets, common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects)
}

func (r *GcpProjectResolver) Resolve(ctx context.Context, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	if tc == nil || tc.Options["project-id"] == "" {
		return resolved, nil
	}

	// Note: we use the resolver instead of the direct gcp_provider.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, tc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	provider, ok := m.Provider.(*gcp_provider.Provider)
	if !ok {
		return nil, errors.New("could not create gcp provider")
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

	project := tc.Options["project-id"]
	var resolvedRoot *asset.Asset
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects) {
		pf.Name = "gcp-project"
		resolvedRoot = &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "GCP project " + project,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			Labels: map[string]string{
				common.ParentId: project,
			},
		}
		resolved = append(resolved, resolvedRoot)
	}

	if tc.IncludesOneOfDiscoveryTarget(append(ProjectDiscoveryTargets, common.DiscoveryAuto, common.DiscoveryAll)...) {
		assetList, err := GatherAssets(ctx, tc, project, credsResolver, sfn)
		if err != nil {
			return nil, err
		}
		for i := range assetList {
			a := assetList[i]
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
}
