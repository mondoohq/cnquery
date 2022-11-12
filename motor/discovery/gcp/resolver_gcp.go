package gcp

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
)

type GcpResolver struct{}

func (k *GcpResolver) Name() string {
	return "GCP Resolver"
}

func (r *GcpResolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects, DiscoveryInstances}
}

func (r *GcpResolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	if tc.Options != nil && (tc.Options["organization"] != "" || tc.Options["organization-id"] != "") {
		// discover the full organization
		return (&GcpOrgResolver{}).Resolve(tc, cfn, sfn, userIdDetectors...)
	} else {
		// when the user has not provided a project, check if we got a project or try to determine it
		if tc.Options == nil || (tc.Options["project"] == "" || tc.Options["project-id"] != "") {
			// try to determine current project
			projectid, err := gcp_provider.GetCurrentProject()
			if err != nil || len(projectid) == 0 {
				return nil, errors.New("gcp: no project id provided")
			}
			if tc.Options == nil {
				tc.Options = map[string]string{}
			}
			tc.Options["project"] = projectid
		}

		// assume it is the local project
		return (&GcpProjectResolver{}).Resolve(tc, cfn, sfn, userIdDetectors...)
	}
}
