package resolver

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/docker"
)

type containerRegistryResolver struct{}

func (k *containerRegistryResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	repository := strings.TrimPrefix(in.Connection, "cr://")
	log.Debug().Str("registry", repository).Msg("fetch meta information from docker registry")
	r := docker.NewDockerRegistryImages()
	// to support self-signed certs
	r.Insecure = opts.Insecure

	assetList, err := r.ListImages(repository)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch container images")
		return nil, err
	}

	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}
