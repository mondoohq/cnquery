package network

import (
	"strconv"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os/fsutil"
)

type Provider struct {
	FQDN    string
	Port    int32
	Scheme  string
	Family  []string
	Options map[string]string
}

func New(conf *providers.Config) (*Provider, error) {
	family := []string{"network"}
	if _, ok := conf.Options["tls"]; ok {
		family = append(family, "tls")
	}

	return &Provider{
		FQDN:    conf.Host,
		Port:    conf.Port,
		Scheme:  conf.Options["scheme"],
		Family:  family,
		Options: conf.Options,
	}, nil
}

func (p *Provider) Identifier() (string, error) {
	host := p.FQDN
	if p.Port != 0 {
		host = p.FQDN + ":" + strconv.Itoa(int(p.Port))
	}

	if _, ok := p.Options["tls"]; ok {
		return "//platformid.api.mondoo.app/runtime/network/tls/" + host, nil
	} else {
		return "//platformid.api.mondoo.app/runtime/network/host/" + host, nil
	}
}

func (p *Provider) URI() string {
	if p.Port == 0 {
		return p.Scheme + "://" + p.FQDN
	}
	return p.Scheme + "://" + p.FQDN + ":" + strconv.Itoa(int(p.Port))
}

func (p *Provider) Supports(mode string) bool {
	for i := range p.Family {
		if p.Family[i] == mode {
			return true
		}
	}
	return false
}

// ----------------- other requirements vv -------------------------

func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, providers.ErrFileInfoNotImplemented
}

func (p *Provider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_NETWORK
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{}
}

func (p *Provider) Runtime() string {
	return ""
}
