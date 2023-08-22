// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package opcua

import (
	"context"
	"errors"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_OPCUA {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	if pCfg.Options == nil || pCfg.Options["endpoint"] == "" {
		return nil, errors.New("opcua provider requires an endpoint. please set option `endpoint`")
	}

	endpoint := pCfg.Options["endpoint"]

	policy := "None" // None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto"
	mode := "None"   //  None, Sign, SignAndEncrypt. Default: auto
	//certFile := "created/server_cert.der"
	//keyFile := "created/server_key.der"

	ctx := context.Background()

	endpoints, err := opcua.GetEndpoints(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	ep := opcua.SelectEndpoint(endpoints, policy, ua.MessageSecurityModeFromString(mode))
	if ep == nil {
		return nil, errors.New("failed to find suitable endpoint")
	}

	opts := []opcua.Option{
		opcua.SecurityPolicy(policy),
		opcua.SecurityModeString(mode),
		//opcua.CertificateFile(certFile),
		//opcua.PrivateKeyFile(keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c := opcua.NewClient(endpoint, opts...)
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	p := &Provider{
		id:       "",
		client:   c,
		endpoint: endpoint,
	}

	return p, nil
}

type Provider struct {
	id       string
	client   *opcua.Client
	endpoint string
}

func (p *Provider) Close() {
	p.client.CloseWithContext(context.Background())
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_OPCUA
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Client() *opcua.Client {
	return p.client
}
