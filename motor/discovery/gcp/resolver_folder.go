package gcp

import (
	"context"
	"fmt"

	"errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

var FolderDiscoveryTargets = append(ProjectDiscoveryTargets)

type GcpFolderResolver struct{}

func (k *GcpFolderResolver) Name() string {
	return "GCP Folder Resolver"
}

func (r *GcpFolderResolver) AvailableDiscoveryTargets() []string {
	return append(FolderDiscoveryTargets, common.DiscoveryAuto, common.DiscoveryAll, DiscoveryFolders)
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

	folderId := tc.Options["folder-id"]
	md, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}

	folder, err := GetValue[string](md, fmt.Sprintf("return gcp.folder(id: '%s').name", folderId))
	if err != nil {
		return nil, err
	}

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
		type project struct {
			Id string
		}
		projects, err := GetList[project](md, fmt.Sprintf("return gcp.folder(id: '%s').projects { id }", folderId))
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
			for i := range assets {
				a := assets[i]
				if resolvedRoot != nil && a.Platform.Name == "gcp-project" {
					a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
				}
				resolved = append(resolved, a)
			}
		}
	}
	return resolved, nil
}
