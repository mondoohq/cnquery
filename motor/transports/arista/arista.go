package arista

import (
	"strconv"

	"github.com/aristanetworks/goeapi"
	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var _ transports.Transport = (*Transport)(nil)
var _ transports.TransportPlatformIdentifier = (*Transport)(nil)

func New(tc *transports.TransportConfig) (*Transport, error) {
	port := goeapi.UseDefaultPortNum
	if len(tc.Port) > 0 {
		p, err := strconv.Atoi(tc.Port)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse port")
		}
		port = p
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
	node, err := goeapi.Connect("https", tc.Host, c.User, string(c.Secret), port)
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
	kind    transports.Kind
	runtime string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("arista does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("arista does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Arista,
	}
}

func (t *Transport) Client() *goeapi.Node {
	return t.node
}

func (t *Transport) Kind() transports.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}

func (t *Transport) GetVersion() (ShowVersion, error) {
	return GetVersion(t.node)
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}
