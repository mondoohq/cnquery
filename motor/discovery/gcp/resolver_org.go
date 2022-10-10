package gcp

import (
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/gcp"
)

type GcpOrgResolver struct{}

func (k *GcpOrgResolver) Name() string {
	return "GCP Organization Resolver"
}

func (r *GcpOrgResolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects}
}

func (r *GcpOrgResolver) Resolve(tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	if tc == nil || tc.Options["organization"] == "" {
		return resolved, nil
	}

	provider, err := gcp_provider.New(tc)
	if err != nil {
		return nil, err
	}

	// TODO: for now we do not add the organization as asset since we need to adapt the polices and queries to distingush
	// between them. Current resources most likely mix with the org, most gcp requests do not work on org level

	//identifier, err := provider.Identifier()
	//if err != nil {
	//	return nil, err
	//}
	//
	//// detect platform info for the asset
	//detector := platform.NewDetector(provider)
	//pf, err := detector.Platform()
	//if err != nil {
	//	return nil, err
	//}
	//
	//resolved = append(resolved, &asset.Asset{
	//	PlatformIds: []string{identifier},
	//	Name:        "GCP organization " + tc.Options["organization"],
	//	Platform:    pf,
	//	Connections: []*transports.TransportConfig{tc}, // pass-in the current config
	//})

	// discover projects
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryProjects) {
		orgId, err := provider.OrganizationID()
		if err != nil {
			return nil, err
		}
		org, err := provider.GetOrganization(orgId)
		if err != nil {
			return nil, err
		}
		projects, err := provider.GetProjectsForOrganization(org)
		if err != nil {
			return nil, err
		}

		for i := range projects {
			project := projects[i]
			projectConfig := tc.Clone()
			projectConfig.Options = map[string]string{
				"project": project.ProjectId,
			}

			assets, err := (&GcpProjectResolver{}).Resolve(projectConfig, cfn, sfn, userIdDetectors...)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, assets...)
		}
	}

	return resolved, nil
}
