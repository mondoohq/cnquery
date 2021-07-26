package gcp

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	gcp_transport "go.mondoo.io/mondoo/motor/transports/gcp"
	"google.golang.org/api/compute/v1"
)

const (
	DiscoveryAll       = "all"
	DiscoveryInstances = "instances"
)

type GcrResolver struct{}

func (r *GcrResolver) Name() string {
	return "GCP Container Registry Resolver"
}

func (r *GcrResolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *GcrResolver) Resolve(t *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	repository := t.Host

	log.Debug().Str("registry", repository).Msg("fetch meta information from gcr registry")
	gcrImages := NewGCRImages()
	assetList, err := gcrImages.ListRepository(repository, true)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch k8s images")
		return nil, err
	}

	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}

type GcpResolver struct{}

func (k *GcpResolver) Name() string {
	return "GCP Resolver"
}

func (r *GcpResolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryInstances}
}

func (r *GcpResolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

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

	project := tc.Options["project"]

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "GCP project " + project,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
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

			// TODO: make this a resolver helper
			for j := range a.Connections {
				conn := a.Connections[j]

				if len(conn.Credentials) == 0 {
					creds, err := sfn(a)
					if err == nil {
						conn.Credentials = []*transports.Credential{creds}
					} else {
						log.Warn().Str("name", a.Name).Msg("could not determine credentials for asset")
					}
				}
			}

			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}
