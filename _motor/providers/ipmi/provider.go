package ipmi

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	impi_client "go.mondoo.com/cnquery/motor/providers/ipmi/client"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg == nil || pCfg.Backend != providers.ProviderType_IPMI {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	port := pCfg.Port
	if port == 0 {
		port = 623
	}

	// search for password secret
	c, err := vault.GetPassword(pCfg.Credentials)
	if err != nil {
		return nil, errors.New("missing password for ipmi provider")
	}

	client, err := impi_client.NewIpmiClient(&impi_client.Connection{
		Hostname:  pCfg.Host,
		Port:      port,
		Username:  c.User,
		Password:  string(c.Secret),
		Interface: "lan",
	})
	if err != nil {
		return nil, err
	}

	err = client.Open()
	if err != nil {
		return nil, err
	}

	return &Provider{
		client: client,
	}, nil
}

type Provider struct {
	client *impi_client.IpmiClient
	guid   string
}

func (p *Provider) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Ipmi,
	}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return ""
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Client() *impi_client.IpmiClient {
	return p.client
}

func (p *Provider) Identifier() (string, error) {
	guid := p.Guid()
	return "//platformid.api.mondoo.app/runtime/ipmi/deviceid/" + guid, nil
}

func (p *Provider) Guid() string {
	if p.guid != "" {
		return p.guid
	}

	resp, err := p.client.DeviceGUID()
	if err != nil {
		log.Error().Err(err).Msg("could not retrieve Ipmi GUID")
	}

	p.guid = resp.GUID
	return p.guid
}
