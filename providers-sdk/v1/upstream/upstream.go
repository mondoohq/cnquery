// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	rangerUtils "go.mondoo.com/cnquery/v10/utils/ranger"
	"go.mondoo.com/ranger-rpc"
	guard_cert_auth "go.mondoo.com/ranger-rpc/plugins/authentication/cert"
	"go.mondoo.com/ranger-rpc/plugins/rangerguard/crypto"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. upstream.proto

const agents_issuer = "mondoo/ams"

func NewServiceAccountRangerPlugin(credentials *ServiceAccountCredentials) (ranger.ClientPlugin, error) {
	if credentials == nil {
		return nil, errors.New("agent credentials must be set")
	}

	// verify that we can read the private key
	privateKey, err := crypto.PrivateKeyFromBytes([]byte(credentials.PrivateKey))
	if err != nil {
		return nil, errors.New("cannot load retrieved key: " + err.Error())
	}

	log.Debug().Str("kid", credentials.Mrn).Str("issuer", agents_issuer).Msg("initialize client authentication")

	// configure authentication plugin, since the server only accepts authenticated calls
	return guard_cert_auth.NewRangerPlugin(guard_cert_auth.ClientConfig{
		PrivateKey: privateKey,
		Issuer:     agents_issuer,
		Kid:        credentials.Mrn,
		Subject:    credentials.Mrn,
	})
}

// mondoo platform config so that resource scan talk upstream
// TODO: this configuration struct does not belong into the MQL package
// nevertheless the MQL runtime needs to have something that allows users
// to store additional credentials so that resource can use those for
// their resources.
type UpstreamClient struct {
	UpstreamConfig
	Plugins    []ranger.ClientPlugin
	HttpClient *http.Client
}

func (c *UpstreamConfig) InitClient(ctx context.Context) (*UpstreamClient, error) {
	certAuth, err := NewServiceAccountRangerPlugin(c.Creds)
	if err != nil {
		return nil, multierr.Wrap(err, "could not initialize client authentication")
	}
	plugins := []ranger.ClientPlugin{certAuth}
	plugins = append(plugins, rangerUtils.DefaultRangerPlugins(cnquery.GetFeatures(ctx))...)

	res := UpstreamClient{
		UpstreamConfig: *c,
		Plugins:        plugins,
		HttpClient:     c.httpClient(),
	}

	return &res, nil
}

func (c *UpstreamConfig) httpClient() *http.Client {
	if c.ApiProxy == "" {
		return ranger.DefaultHttpClient()
	}
	proxy, err := url.Parse(c.ApiProxy)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse proxy URL")
	}
	return ranger.NewHttpClient(ranger.WithProxy(proxy))
}
