package vcd

import (
	"fmt"
	"net/url"

	"errors"
	"github.com/rs/zerolog/log"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

type vcdConfig struct {
	User     string
	Password string
	Host     string
	Org      string
	Insecure bool
}

func (c *vcdConfig) Href() string {
	return fmt.Sprintf("https://%s/api", c.Host)
}

func newVcdClient(c *vcdConfig) (*govcd.VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href())
	if err != nil {
		return nil, fmt.Errorf("unable to pass url: %s", err)
	}

	vcdClient := govcd.NewVCDClient(*u, c.Insecure)

	err = vcdClient.Authenticate(c.User, c.Password, c.Org)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate: %s", err)
	}
	return vcdClient, nil
}

func New(pCfg *providers.Config) (*Provider, error) {
	if len(pCfg.Credentials) == 0 {
		return nil, errors.New("missing credentials for VMware Cloud Director")
	}

	cfg := &vcdConfig{
		Host:     pCfg.Host,
		Insecure: pCfg.Insecure,
	}

	// determine the organization for the user
	org, ok := pCfg.Options["organization"]
	if ok {
		cfg.Org = org
	} else {
		cfg.Org = "system" // default in vcd
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	if len(pCfg.Credentials) > 0 {
		for i := range pCfg.Credentials {
			cred := pCfg.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				cfg.User = cred.User
				cfg.Password = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for VMware Cloud Director provider")
			}
		}
	}

	client, err := newVcdClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client: client,
		host:   pCfg.Host,
		opts:   pCfg.Options,
	}, nil
}

type Provider struct {
	client *govcd.VCDClient
	host   string
	opts   map[string]string
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

func (p *Provider) Client() *govcd.VCDClient {
	return p.client
}
