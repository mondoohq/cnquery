package gcp

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"google.golang.org/api/compute/v1"
)

type GcpProjectResolver struct{}

func (k *GcpProjectResolver) Name() string {
	return "GCP Project Resolver"
}

func (r *GcpProjectResolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects, DiscoveryInstances}
}

func (r *GcpProjectResolver) Resolve(tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// FIXME: DEPRECATED, update in v8.0 vv
	// The option "project" has been deprecated in favor of project-id
	if tc == nil || (tc.Options["project"] == "" && tc.Options["project-id"] == "") {
		// ^^
		return resolved, nil
	}

	provider, err := gcp_provider.New(tc)
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

	project := tc.Options["project-id"]
	// FIXME: DEPRECATED, remove in v8.0 vv
	// The option "project" has been deprecated in favor of project-id
	if project == "" {
		project = tc.Options["project"]
	}
	// ^^

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProjects) {
		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "GCP project " + project,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			Labels: map[string]string{
				common.ParentId: project,
			},
		})
	}

	// discover compute instances
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		client, err := provider.Client(compute.ComputeReadonlyScope)
		if err != nil {
			return nil, errors.Wrap(err, "use `gcloud auth application-default login` to authenticate locally")
		}

		compute := NewCompute(client)
		compute.Insecure = tc.Insecure

		assetList, err := compute.ListInstancesInProject(project)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch gcp compute instances")
		}
		log.Debug().Int("instances", len(assetList)).Msg("completed instance search")

		for i := range assetList {
			a := assetList[i]
			log.Debug().Str("name", a.Name).Msg("resolved gcp compute instance")

			// find the secret reference for the asset
			common.EnrichAssetWithSecrets(a, sfn)

			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}
