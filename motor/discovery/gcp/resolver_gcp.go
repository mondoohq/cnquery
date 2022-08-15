package gcp

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
	gcp_transport "go.mondoo.io/mondoo/motor/providers/gcp"
)

type GcpResolver struct{}

func (k *GcpResolver) Name() string {
	return "GCP Resolver"
}

func (r *GcpResolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryProjects, DiscoveryInstances}
}

func (r *GcpResolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	if tc.Options != nil && tc.Options["organization"] != "" {
		// discover the full organization
		return (&GcpOrgResolver{}).Resolve(tc, cfn, sfn, userIdDetectors...)
	} else {
		// when the user has not provided a project, check if we got a project or try to determine it
		if tc.Options == nil || tc.Options["project"] == "" {
			// try to determine current project
			projectid, err := gcp_transport.GetCurrentProject()
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
