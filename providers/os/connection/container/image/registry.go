// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package image

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/auth"
)

// Option is a functional option
// see https://www.sohamkamani.com/golang/options-pattern/
type Option func(*options) error

type options struct {
	insecure bool
	auth     authn.Authenticator
}

func WithInsecure(insecure bool) Option {
	return func(o *options) error {
		o.insecure = insecure
		return nil
	}
}

func WithAuthenticator(auth authn.Authenticator) Option {
	return func(o *options) error {
		o.auth = auth
		return nil
	}
}

func AuthOption(credentials []*vault.Credential) []Option {
	remoteOpts := []Option{}
	for i := range credentials {
		cred := credentials[i]
		switch cred.Type {
		case vault.CredentialType_password:
			log.Debug().Msg("add password authentication")
			cfg := authn.AuthConfig{
				Username: cred.User,
				Password: string(cred.Secret),
			}
			remoteOpts = append(remoteOpts, WithAuthenticator((authn.FromConfig(cfg))))
		case vault.CredentialType_bearer:
			log.Debug().Str("token", string(cred.Secret)).Msg("add bearer authentication")
			cfg := authn.AuthConfig{
				Username:      cred.User,
				RegistryToken: string(cred.Secret),
			}
			remoteOpts = append(remoteOpts, WithAuthenticator((authn.FromConfig(cfg))))
		default:
			log.Warn().Msg("unknown credentials for container image")
			logger.DebugJSON(credentials)
		}
	}
	return remoteOpts
}

func DefaultAuthOpts(ref name.Reference) (authn.Authenticator, error) {
	kc := auth.ConstructKeychain(ref.Name())
	return kc.Resolve(ref.Context())
}

func GetImageDescriptor(ref name.Reference, opts ...Option) (*remote.Descriptor, error) {
	o := &options{
		insecure: false,
	}

	for _, option := range opts {
		if err := option(o); err != nil {
			return nil, err
		}
	}

	if o.auth == nil {
		auth, err := DefaultAuthOpts(ref)
		if err != nil {
			return nil, err
		}
		o.auth = auth
	}

	return remote.Get(ref, remote.WithAuth(o.auth))
}

func LoadImageFromRegistry(ref name.Reference, opts ...Option) (v1.Image, error) {
	o := &options{
		insecure: false,
	}

	for _, option := range opts {
		if err := option(o); err != nil {
			return nil, err
		}
	}

	if o.auth == nil {
		auth, err := DefaultAuthOpts(ref)
		if err != nil {
			return nil, err
		}
		o.auth = auth
	}

	// mimic http.DefaultTransport
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if o.insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	img, err := remote.Image(ref, remote.WithAuth(o.auth), remote.WithTransport(tr))
	if err != nil {
		return nil, err
	}
	return img, nil
}
