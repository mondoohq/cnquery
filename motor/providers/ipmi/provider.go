package ipmi

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi/resources/ipmi"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var (
	_ providers.Transport                   = (*Provider)(nil)
	_ providers.TransportPlatformIdentifier = (*Provider)(nil)
)

func New(tc *providers.TransportConfig) (*Provider, error) {
	if tc == nil || tc.Backend != providers.ProviderType_IPMI {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	port := tc.Port
	if port == 0 {
		port = 623
	}

	// search for password secret
	c, err := vault.GetPassword(tc.Credentials)
	if err != nil {
		return nil, errors.New("missing password for ipmi provider")
	}

	client, err := ipmi.NewIpmiClient(&ipmi.Connection{
		Hostname:  tc.Host,
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
	client *ipmi.IpmiClient
	guid   string
}

func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, providers.ErrFileInfoNotImplemented
}

func (p *Provider) FS() afero.Fs {
	return &fsutil.NoFs{}
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

func (p *Provider) Client() *ipmi.IpmiClient {
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
