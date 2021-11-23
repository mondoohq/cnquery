package ipmi

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi/resources/ipmi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
)

var _ transports.Transport = (*Transport)(nil)
var _ transports.TransportIdentifier = (*Transport)(nil)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc == nil || tc.Backend != transports.TransportBackend_CONNECTION_IPMI {
		return nil, errors.New("backend is not supported for ipmi transport")
	}

	port, err := tc.IntPort()
	if err != nil {
		return nil, errors.New("port is not a valid number " + tc.Port)
	}

	// fallback to default port 623
	if port == 0 {
		port = 623
	}

	// search for password secret
	c, err := vault.GetPassword(tc.Credentials)
	if err != nil {
		return nil, errors.New("missing password for ipmi transport")
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

	return &Transport{
		client: client,
	}, nil
}

type Transport struct {
	client *ipmi.IpmiClient
	guid   string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("ipmi does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("ipmi does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {
	if t.client != nil {
		t.client.Close()
	}
}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Ipmi,
	}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportIdentifierDetector,
	}
}

func (t *Transport) Client() *ipmi.IpmiClient {
	return t.client
}

func (t *Transport) Identifier() (string, error) {
	guid := t.Guid()
	return "//platformid.api.mondoo.app/runtime/ipmi/deviceid/" + guid, nil
}

func (t *Transport) Guid() string {
	if t.guid != "" {
		return t.guid
	}

	resp, err := t.client.DeviceGUID()
	if err != nil {
		log.Error().Err(err).Msg("could not retrieve Ipmi GUID")
	}

	t.guid = resp.GUID
	return t.guid
}
