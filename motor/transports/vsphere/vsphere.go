package vsphere

import (
	"context"
	"errors"
	"net/url"

	"github.com/spf13/afero"
	"github.com/vmware/govmomi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var _ transports.Transport = (*Transport)(nil)
var _ transports.TransportPlatformIdentifier = (*Transport)(nil)

func VSphereConnectionURL(hostname string, port string, user string, password string) (*url.URL, error) {
	host := hostname
	if len(port) > 0 {
		host = hostname + ":" + port
	}

	u, err := url.Parse("https://" + host + "/sdk")
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(user, password)
	return u, nil
}

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_VSPHERE {
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
	kind               transports.Kind
	runtime            string
	opts               map[string]string
	selectedPlatformID string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("vsphere does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("vsphere does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_vSphere,
	}
}

func (t *Transport) Client() *govmomi.Client {
	return t.client
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}
