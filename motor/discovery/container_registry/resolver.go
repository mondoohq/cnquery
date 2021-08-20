package container_registry

import (
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

type Resolver struct {
	// NoStrictValidation deactivates the strict validation for container registry resolutions
	// cr://index.docker.io/mondoolabs/mondoo would be converted index.docker.io/mondoolabs/mondoo:latest
	// It is not the default behavior but is used by the docker resolver to resolve images
	NoStrictValidation bool
}

func (r *Resolver) Name() string {
	return "Container Registry Discover"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	imageFetcher := NewContainerRegistry()
	// to support self-signed certs
	imageFetcher.Insecure = tc.Insecure

	// check if the reference is an image
	// NOTE: we use strict validation here otherwise urls like cr://index.docker.io/mondoolabs/mondoo are converted
	// to index.docker.io/mondoolabs/mondoo:latest
	opts := name.StrictValidation
	if r.NoStrictValidation {
		opts = name.WeakValidation
	}

	ref, err := name.ParseReference(tc.Host, opts)
	if err == nil {
		log.Debug().Str("image", tc.Host).Msg("detected container image in container registry")

		remoteOpts := AuthOption(tc.Credentials, cfn)
		a, err := imageFetcher.GetImage(ref, tc.Credentials, remoteOpts...)
		if err != nil {
			return nil, err
		}

		if tc.Insecure {
			for i := range a.Connections {
				c := a.Connections[i]
				c.Insecure = tc.Insecure
				c.Credentials = tc.Credentials
			}
		}

		return []*asset.Asset{a}, nil
	}

	// okay, no image, lets check the repository
	repository := tc.Host
	log.Info().Str("registry", repository).Msg("fetch meta information from container registry")

	assetList, err := imageFetcher.ListImages(repository)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch container images")
		return nil, err
	}

	for i := range assetList {
		a := assetList[i]
		log.Info().Str("name", a.Name).Str("image", a.Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")

		if tc.Insecure {
			for i := range a.Connections {
				c := a.Connections[i]
				c.Insecure = tc.Insecure
			}
		}
		resolved = append(resolved, a)
	}

	if len(resolved) == 0 {
		return nil, errors.New("could not find repository:" + repository)
	}

	return resolved, nil
}

func AuthOption(credentials []*vault.Credential, cfn common.CredentialFn) []remote.Option {
	remoteOpts := []remote.Option{}
	for i := range credentials {
		cred := credentials[i]

		// NOTE: normally the motor connection is resolving the credentials but here we need the credential earlier
		// we probably want to write some mql resources to support the query of registries itself
		resolvedCredential, err := cfn(cred)
		if err != nil {
			log.Warn().Err(err).Msg("could not resolve credential")
		}
		switch resolvedCredential.Type {
		case vault.CredentialType_password:
			log.Debug().Msg("add password authentication")
			cfg := authn.AuthConfig{
				Username: resolvedCredential.User,
				Password: string(resolvedCredential.Secret),
			}
			remoteOpts = append(remoteOpts, remote.WithAuth(authn.FromConfig(cfg)))
		case vault.CredentialType_bearer:
			log.Debug().Str("token", string(resolvedCredential.Secret)).Msg("add bearer authentication")
			cfg := authn.AuthConfig{
				Username:      resolvedCredential.User,
				RegistryToken: string(resolvedCredential.Secret),
			}
			remoteOpts = append(remoteOpts, remote.WithAuth(authn.FromConfig(cfg)))
		default:
			log.Warn().Msg("unknown credentials for container image")
			logger.DebugJSON(credentials)
		}
	}
	return remoteOpts
}
