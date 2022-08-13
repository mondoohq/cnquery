package vsphere

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/spf13/afero"
	"github.com/vmware/govmomi"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var (
	_ providers.Transport                   = (*Provider)(nil)
	_ providers.TransportPlatformIdentifier = (*Provider)(nil)
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

func New(tc *providers.TransportConfig) (*Provider, error) {
	if tc.Backend != providers.ProviderType_VSPHERE {
		return nil, errors.New("backend is not supported for vSphere transport")
	}

	// search for password secret
	c, err := vault.GetPassword(tc.Credentials)
	if err != nil {
		return nil, errors.New("missing password for vSphere transport")
	}

	// derive vsphere connection url from Provider Config
	vsphereUrl, err := VSphereConnectionURL(tc.Host, tc.Port, c.User, string(c.Secret))
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
		kind:               tc.Kind,
		runtime:            tc.Runtime,
		opts:               tc.Options,
		selectedPlatformID: tc.PlatformId,
	}, nil
}

type Provider struct {
	client             *govmomi.Client
	kind               providers.Kind
	runtime            string
	opts               map[string]string
	selectedPlatformID string
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
