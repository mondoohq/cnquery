package sample

import (
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	return &Provider{
		opts: pCfg.Options,
	}, nil
}

type Provider struct {
	opts map[string]string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return "vcd"
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	return &platform.Platform{
		Name:    "sample",
		Title:   "Sample",
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
		Labels: map[string]string{
			"sample.com/api-version": "v1",
		},
	}, nil
}

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/sample/host/sample", nil
}
