package arista

import (
	"github.com/aristanetworks/goeapi"
	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var (
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(tc *providers.TransportConfig) (*Transport, error) {
	port := tc.Port
	if port == 0 {
		port = goeapi.UseDefaultPortNum
	}

	if len(tc.Credentials) == 0 {
		return nil, errors.New("missing password for arista transport")
	}

	// search for password secret
	c, err := vault.GetPassword(tc.Credentials)
	if err != nil {
		return nil, errors.New("missing password for arista transport")
	}

	// NOTE: we explicitly do not support http, since there is no real reason to support http
	// the goeapi is always running in insecure mode since it does not verify the server
	// setup which allows potential man-in-the-middle attacks, consider opening a PR
	// https://github.com/aristanetworks/goeapi/blob/7944bcedaf212bb60e5f9baaf471469f49113f47/eapilib.go#L527
	node, err := goeapi.Connect("https", tc.Host, c.User, string(c.Secret), int(port))
	if err != nil {
		return nil, err
	}

	return &Transport{
		node:    node,
		kind:    tc.Kind,
		runtime: tc.Runtime,
	}, nil
}

type Transport struct {
	node    *goeapi.Node
	kind    providers.Kind
	runtime string
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("arista does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("arista does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Arista,
	}
}

func (t *Transport) Client() *goeapi.Node {
	return t.node
}

func (t *Transport) Kind() providers.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}

func (t *Transport) GetVersion() (ShowVersion, error) {
	return GetVersion(t.node)
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
