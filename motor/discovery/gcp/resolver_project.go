package gcp

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers"
	gcp_transport "go.mondoo.io/mondoo/motor/providers/gcp"
	"google.golang.org/api/compute/v1"
)

type GcpProjectResolver struct{}

func (k *GcpProjectResolver) Name() string {
	return "GCP Project Resolver"
}

func (r *GcpProjectResolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryInstances}
}

func (r *GcpProjectResolver) Resolve(tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	if tc == nil || tc.Options["project"] == "" {
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
	detector := detector.New(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	project := tc.Options["project"]

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "GCP project " + project,
		Platform:    pf,
		Connections: []*providers.Config{tc}, // pass-in the current config
		Labels: map[string]string{
			common.ParentId: project,
		},
	})

	// discover compute instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryInstances) {
		client, err := trans.Client(compute.ComputeReadonlyScope)
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
