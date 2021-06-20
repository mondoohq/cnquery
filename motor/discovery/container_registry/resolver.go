package container_registry

import (
	"errors"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"go.mondoo.io/mondoo/motor/platform"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Container Registry Discover"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	repository := strings.TrimPrefix(url, "cr://")

	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY,
		Host:    repository,
	}

	for i := range opts {
		opts[i](tc)
	}
	return tc, nil
}

func (r *Resolver) Resolve(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// check if the reference is an image
	// NOTE: we use strict validation here otherwise urls like cr://index.docker.io/mondoolabs/mondoo are converted
	// to index.docker.io/mondoolabs/mondoo:latest
	_, err := name.ParseReference(tc.Host, name.StrictValidation)
	if err == nil {
		log.Debug().Str("image", tc.Host).Msg("detected container image in registry")
		// TODO: sync implementation with docker resolver image
		tc.Backend = transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY
		resolved = append(resolved, &asset.Asset{
			Name: tc.Host,
			// PlatformIDs: []string{}, // we cannot determine the id here
			Connections: []*transports.TransportConfig{tc},
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: transports.RUNTIME_DOCKER_REGISTRY,
			},
		})
		return resolved, nil
	}

	// okay, no image, lets check the repository
	repository := tc.Host
	log.Info().Str("registry", repository).Msg("fetch meta information from docker registry")
	imageFetcher := NewDockerRegistryImages()
	// to support self-signed certs
	imageFetcher.Insecure = tc.Insecure

	assetList, err := imageFetcher.ListImages(repository)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch container images")
		return nil, err
	}

	for i := range assetList {
		log.Info().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	if len(resolved) == 0 {
		return nil, errors.New("could not find repository:" + repository)
	}

	return resolved, nil
}
