package oci

import (
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	configProvider := common.DefaultConfigProvider()

	tenancyOcid, err := configProvider.TenancyOCID()
	if err != nil {
		return nil, err
	}

	t := &Provider{
		// opts:   pCfg.Options,
		config: configProvider,

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
