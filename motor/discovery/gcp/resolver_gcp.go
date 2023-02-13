package gcp

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/vault"
)

type GcpResolver struct{}

func (k *GcpResolver) Name() string {
	return "GCP Resolver"
}

func (r *GcpResolver) AvailableDiscoveryTargets() []string {
	return []string{
		common.DiscoveryAuto, common.DiscoveryAll, DiscoveryOrganization, DiscoveryFolders, DiscoveryProjects,
		DiscoveryInstances, DiscoveryComputeImages, DiscoveryComputeNetworks, DiscoveryComputeSubnetworks, DiscoveryComputeFirewalls,
		DiscoveryGkeClusters,
		DiscoveryStorageBuckets,
		DiscoveryBigQueryDatasets,
	}
}

func (r *GcpResolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	// FIXME: DEPRECATED, update in v8.0 vv
	// The option "organization" has been deprecated in favor of organization-id
	if tc.Options != nil && (tc.Options["organization"] != "" || tc.Options["organization-id"] != "") {
		// ^^
		// discover the full organization
		return (&GcpOrgResolver{}).Resolve(ctx, tc, credsResolver, sfn, userIdDetectors...)
	} else {
		// when the user has not provided a project, check if we got a project or try to determine it
		// FIXME: DEPRECATED, update in v8.0 vv
		// The option "project" has been deprecated in favor of project-id
		if tc.Options == nil || (tc.Options["project"] == "" && tc.Options["project-id"] == "") {
			// ^^
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
		return (&GcpProjectResolver{}).Resolve(ctx, tc, credsResolver, sfn, userIdDetectors...)
	}
}
