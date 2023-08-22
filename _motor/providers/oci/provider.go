// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	var configProvider common.ConfigurationProvider
	// if we have passed in credentials, assume we want to pass in all values explicitly.
	if len(pCfg.Credentials) > 0 {
		fingerprint := pCfg.Options["fingerprint"]
		if fingerprint == "" {
			return nil, errors.New("OCI provider fingerprint value cannot be empty")
		}
		tenancyOcid := pCfg.Options["tenancy"]
		if tenancyOcid == "" {
			return nil, errors.New("OCI provider tenancy value cannot be empty")
		}
		userOcid := pCfg.Options["user"]
		if userOcid == "" {
			return nil, errors.New("OCI provider user value cannot be empty")
		}
		region := pCfg.Options["region"]
		if region == "" {
			return nil, errors.New("OCI provider region value cannot be empty")
		}
		pkey := pCfg.Credentials[0]
		if pkey.Type != vault.CredentialType_private_key {
			return nil, errors.New("OCI provider does not support credential type: " + pkey.Type.String())
		}
		configProvider = common.NewRawConfigurationProvider(tenancyOcid, userOcid, region, fingerprint, string(pkey.Secret), nil)
	} else {
		configProvider = common.DefaultConfigProvider()
	}
	tenancyOcid, err := configProvider.TenancyOCID()
	if err != nil {
		return nil, err
	}

	t := &Provider{
		// opts:   pCfg.Options,
		config:      configProvider,
		tenancyOcid: tenancyOcid,
	}

	return t, nil
}

type Provider struct {
	id   string
	opts map[string]string

	config      common.ConfigurationProvider
	tenancyOcid string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Google,
	}
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_OCI
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
