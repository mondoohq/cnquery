package gcp

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
)

type GcrResolver struct{}

func (r *GcrResolver) Name() string {
	return "GCP Container Registry Resolver"
}

func (r *GcrResolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *GcrResolver) Resolve(root *asset.Asset, t *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
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
