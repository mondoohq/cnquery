// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package auth

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/os/connection/container/image"
)

func AuthOption(credentials []*vault.Credential) []image.Option {
	remoteOpts := []image.Option{}
	for i := range credentials {
		cred := credentials[i]
		switch cred.Type {
		case vault.CredentialType_password:
			log.Debug().Msg("add password authentication")
			cfg := authn.AuthConfig{
				Username: cred.User,
				Password: string(cred.Secret),
			}
			remoteOpts = append(remoteOpts, image.WithAuthenticator((authn.FromConfig(cfg))))
		case vault.CredentialType_bearer:
			log.Debug().Str("token", string(cred.Secret)).Msg("add bearer authentication")
			cfg := authn.AuthConfig{
				Username:      cred.User,
				RegistryToken: string(cred.Secret),
			}
			remoteOpts = append(remoteOpts, image.WithAuthenticator((authn.FromConfig(cfg))))
		default:
			log.Warn().Msg("unknown credentials for container image")
			logger.DebugJSON(credentials)
		}
	}
	return remoteOpts
}
