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
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
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

func New(tc *providers.TransportConfig) (*Transport, error) {
	if tc.Backend != providers.TransportBackend_CONNECTION_VSPHERE {
		return nil, errors.New("backend is not supported for vSphere transport")
	}

	// search for password secret
	c, err := vault.GetPassword(tc.Credentials)
	if err != nil {
		return nil, errors.New("missing password for vSphere transport")
	}

	// derive vsphere connection url from Transport Config
	vsphereUrl, err := VSphereConnectionURL(tc.Host, tc.Port, c.User, string(c.Secret))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, vsphereUrl, true)
	if err != nil {
		return nil, err
	}

	return &Transport{
		client:             client,
		kind:               tc.Kind,
		runtime:            tc.Runtime,
		opts:               tc.Options,
		selectedPlatformID: tc.PlatformId,
	}, nil
}

type Transport struct {
	client             *govmomi.Client
	kind               providers.Kind
	runtime            string
	opts               map[string]string
	selectedPlatformID string
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("vsphere does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("vsphere does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_vSphere,
	}
}

func (t *Transport) Client() *govmomi.Client {
	return t.client
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() providers.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
