package gcp

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/vault"
)

var OrgDiscoveryTargets = append(FolderDiscoveryTargets, DiscoveryOrganization, DiscoveryFolders, DiscoveryProjects)

type GcpOrgResolver struct{}

func (k *GcpOrgResolver) Name() string {
	return "GCP Organization Resolver"
}

func (r *GcpOrgResolver) AvailableDiscoveryTargets() []string {
	return append(OrgDiscoveryTargets, common.DiscoveryAuto, common.DiscoveryAll)
}

func (r *GcpOrgResolver) Resolve(ctx context.Context, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// FIXME: DEPRECATED, update in v8.0 vv
	// The option "organization" has been deprecated in favor of organization-id
	if tc == nil || (tc.Options["organization"] == "" && tc.Options["organization-id"] == "") {
		// ^^
		return resolved, nil
	}

	provider, err := gcp_provider.New(tc)
	if err != nil {
		return nil, err
	}

	orgId, err := provider.OrganizationID()
	if err != nil {
		return nil, err
	}
	org, err := provider.GetOrganization(orgId)
	if err != nil {
		return nil, err
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

	var rootAsset *asset.Asset
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryOrganization) {
		pf.Name = "gcp-org"
		rootAsset = &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "GCP organization " + org.DisplayName,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
		}
		resolved = append(resolved, rootAsset)
	}

	// discover folders
	if tc.IncludesOneOfDiscoveryTarget(DiscoveryFolders) {
		m, err := NewMQLAssetsDiscovery(provider)
		if err != nil {
			return nil, err
		}

		type folder struct {
			Id string
		}
		folders, err := GetList[folder](m, "return gcp.organization.folders { id }")
		if err != nil {
			return nil, err
		}

		for _, f := range folders {
			folderConfig := tc.Clone()
			folderConfig.Options = map[string]string{
				"folder-id": f.Id,
			}

			assets, err := (&GcpFolderResolver{}).Resolve(ctx, folderConfig, credsResolver, sfn, userIdDetectors...)
			if err != nil {
				return nil, err
			}
			for i := range assets {
				a := assets[i]
				if rootAsset != nil {
					a.RelatedAssets = append(a.RelatedAssets, rootAsset)
				}
				resolved = append(resolved, a)
			}
			resolved = append(resolved, assets...)
		}
	}

	// discover projects
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryProjects) {
		m, err := NewMQLAssetsDiscovery(provider)
		if err != nil {
			return nil, err
		}

		type project struct {
			Id string
		}
		projects, err := GetList[project](m, "return gcp.organization.projects { id }")
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
				if rootAsset != nil {
					a.RelatedAssets = append(a.RelatedAssets, rootAsset)
				}
				resolved = append(resolved, a)
			}
			resolved = append(resolved, assets...)
		}
	}

	return resolved, nil
}
