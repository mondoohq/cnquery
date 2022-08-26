package vsphere

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/vmware/govmomi"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func VSphereConnectionURL(hostname string, port int32, user string, password string) (*url.URL, error) {
	host := hostname
	if port > 0 {
		host = hostname + ":" + strconv.Itoa(int(port))
	}

	u, err := url.Parse("https://" + host + "/sdk")
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(user, password)
	return u, nil
}

func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_VSPHERE {
		return nil, errors.New("backend is not supported for vSphere transport")
	}

	// search for password secret
	c, err := vault.GetPassword(pCfg.Credentials)
	if err != nil {
		return nil, errors.New("missing password for vSphere transport")
	}

	// derive vsphere connection url from Provider Config
	vsphereUrl, err := VSphereConnectionURL(pCfg.Host, pCfg.Port, c.User, string(c.Secret))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, vsphereUrl, true)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:             client,
		kind:               pCfg.Kind,
		runtime:            pCfg.Runtime,
		opts:               pCfg.Options,
		selectedPlatformID: pCfg.PlatformId,
	}, nil
}

type Provider struct {
	client             *govmomi.Client
	kind               providers.Kind
	runtime            string
	opts               map[string]string
	selectedPlatformID string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_vSphere,
	}
}

func (p *Provider) Client() *govmomi.Client {
	return p.client
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return p.kind
}

func (p *Provider) Runtime() string {
	return p.runtime
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
