package gcp

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

type GcpFolderResolver struct{}

func (k *GcpFolderResolver) Name() string {
	return "GCP Folder Resolver"
}

func (r *GcpFolderResolver) AvailableDiscoveryTargets() []string {
	return []string{
		common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects,
		DiscoveryInstances, DiscoveryComputeImages, DiscoveryComputeNetworks, DiscoveryComputeSubnetworks, DiscoveryComputeFirewalls,
		DiscoveryGkeClusters,
		DiscoveryStorageBuckets,
		DiscoveryBigQueryDatasets,
	}
}

func (r *GcpFolderResolver) Resolve(ctx context.Context, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	if tc == nil || tc.Options["folder-id"] == "" {
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

	folder := tc.Options["folder-id"]

	var resolvedRoot *asset.Asset
	if tc.IncludesOneOfDiscoveryTarget(DiscoveryFolders) {
		pf.Name = "gcp-folder"
		resolvedRoot = &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "GCP folder " + folder,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
		}
		resolved = append(resolved, resolvedRoot)
	}

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll,
		DiscoveryInstances, DiscoveryComputeImages, DiscoveryComputeNetworks, DiscoveryComputeSubnetworks, DiscoveryComputeFirewalls,
		DiscoveryGkeClusters,
		DiscoveryStorageBuckets,
		DiscoveryBigQueryDatasets) {
		m, err := NewMQLAssetsDiscovery(provider)
		if err != nil {
			return nil, err
		}

		type project struct {
			Id string
		}
		projects, err := GetList[project](m, fmt.Sprintf("return gcp.folder(id: '%s').projects.all { id }", folder))
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			projectConfig := tc.Clone()
			projectConfig.Options = map[string]string{
				"project-id": p.Id,
			}

			assets, err := (&GcpProjectResolver{}).Resolve(ctx, projectConfig, credsResolver, sfn, userIdDetectors...)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, assets...)
		}
	}
	return resolved, nil
}
