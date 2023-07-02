package arista

import (
	"github.com/aristanetworks/goeapi"
	"errors"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	port := pCfg.Port
	if port == 0 {
		port = goeapi.UseDefaultPortNum
	}

	if len(pCfg.Credentials) == 0 {
		return nil, errors.New("missing password for arista provider")
	}

	// search for password secret
	c, err := vault.GetPassword(pCfg.Credentials)
	if err != nil {
		return nil, errors.New("missing password for arista provider")
	}

	// NOTE: we explicitly do not support http, since there is no real reason to support http
	// the goeapi is always running in insecure mode since it does not verify the server
	// setup which allows potential man-in-the-middle attacks, consider opening a PR
	// https://github.com/aristanetworks/goeapi/blob/7944bcedaf212bb60e5f9baaf471469f49113f47/eapilib.go#L527
	node, err := goeapi.Connect("https", pCfg.Host, c.User, string(c.Secret), int(port))
	if err != nil {
		return nil, err
	}

	return &Provider{
		node:    node,
		kind:    pCfg.Kind,
		runtime: pCfg.Runtime,
	}, nil
}

type Provider struct {
	node    *goeapi.Node
	kind    providers.Kind
	runtime string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Arista,
	}
}

func (p *Provider) Client() *goeapi.Node {
	return p.node
}

func (p *Provider) Kind() providers.Kind {
	return p.kind
}

func (p *Provider) Runtime() string {
	return p.runtime
}

func (p *Provider) GetVersion() (ShowVersion, error) {
	return GetVersion(p.node)
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
