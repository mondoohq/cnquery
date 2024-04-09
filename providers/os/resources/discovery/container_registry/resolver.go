// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package container_registry

import (
	"context"
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

type Resolver struct {
	// NoStrictValidation deactivates the strict validation for container registry resolutions
	// cr://index.docker.io/mondoo/client would be converted index.docker.io/mondoo/client:latest
	// It is not the default behavior but is used by the docker resolver to resolve images
	NoStrictValidation bool
}

func (r *Resolver) Name() string {
	return "Container Registry Discover"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{"auto", "all"}
}

func (r *Resolver) Resolve(ctx context.Context, root *inventory.Asset, conf *inventory.Config, credsResolver vault.Resolver) ([]*inventory.Asset, error) {
	resolved := []*inventory.Asset{}

	imageFetcher := NewContainerRegistryResolver()
	// to support self-signed certs
	imageFetcher.Insecure = conf.Insecure

	// check if the reference is an image
	// NOTE: we use strict validation here otherwise urls like cr://index.docker.io/mondoo/client are converted
	// to index.docker.io/mondoo/client:latest
	opts := name.StrictValidation
	if r.NoStrictValidation {
		opts = name.WeakValidation
	}

	ref, err := name.ParseReference(conf.Host, opts)
	if err == nil {
		log.Debug().Str("image", conf.Host).Msg("detected container image in container registry")

		remoteOpts := AuthOption(conf.Credentials, credsResolver)
		// we need to disable default keychain auth if an auth method was found
		if len(remoteOpts) > 0 {
			imageFetcher.DisableKeychainAuth = true
		}

		a, err := imageFetcher.GetImage(ref, conf.Credentials, remoteOpts...)
		if err != nil {
			return nil, err
		}
		// keep already set options, i.e. image paths
		if conf.Options != nil && a.Connections[0].Options == nil {
			a.Connections[0].Options = conf.Options
		}

		if conf.Insecure {
			for i := range a.Connections {
				c := a.Connections[i]
				c.Insecure = conf.Insecure
				c.Credentials = conf.Credentials
			}
		}

		return []*inventory.Asset{a}, nil
	}

	// okay, no image, lets check the repository
	repository := conf.Host
	log.Info().Str("registry", repository).Msg("fetch meta information from container registry")

	assetList, err := imageFetcher.ListImages(repository)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch container images")
		return nil, err
	}

	for i := range assetList {
		a := assetList[i]
		log.Info().Str("name", a.Name).Str("image", a.Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")

		if conf.Insecure {
			for i := range a.Connections {
				c := a.Connections[i]
				c.Insecure = conf.Insecure
			}
		}
		resolved = append(resolved, a)
	}

	if len(resolved) == 0 {
		return nil, errors.New("could not find repository:" + repository)
	}

	return resolved, nil
}

func AuthOption(credentials []*vault.Credential, credsResolver vault.Resolver) []remote.Option {
	remoteOpts := []remote.Option{}
	for i := range credentials {
		cred := credentials[i]

		// NOTE: normally the motor connection is resolving the credentials but here we need the credential earlier
		// we probably want to write some mql resources to support the query of registries itself
		resolvedCredential, err := credsResolver.GetCredential(cred)
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
