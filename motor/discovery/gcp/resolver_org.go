package gcp

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	gcp_transport "go.mondoo.io/mondoo/motor/transports/gcp"
)

type GcpOrgResolver struct{}

func (k *GcpOrgResolver) Name() string {
	return "GCP Organization Resolver"
}

func (r *GcpOrgResolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryProjects}
}

func (r *GcpOrgResolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	if tc == nil || tc.Options["organization"] == "" {
		return resolved, nil
	}

	trans, err := gcp_transport.New(tc)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	organization := tc.Options["organization"]

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "GCP organization " + organization,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
	})

	// discover projects
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryProjects) {
		orgId, err := trans.OrganizationID()
		if err != nil {
			return nil, err
		}
		org, err := trans.GetOrganization(orgId)
		if err != nil {
			return nil, err
		}
		projects, err := trans.GetProjectsForOrganization(org)
		if err != nil {
			return nil, err
		}

		for i := range projects {
			project := projects[i]
			projectConfig := tc.Clone()
			projectConfig.Options = map[string]string{
				"project": project.ProjectId,
			}
			resolved = append(resolved, &asset.Asset{
				PlatformIds: []string{identifier},
				Name:        "GCP project " + project.ProjectId,
				Platform:    pf,
				Connections: []*transports.TransportConfig{projectConfig}, // pass-in the current config
			})
		}
	}

	return resolved, nil
}
