package ipmi

import (
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi/resources/ipmi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc == nil || tc.Backend != transports.TransportBackend_CONNECTION_IPMI {
		return nil, errors.New("backend is not supported for ipmi transport")
	}

	// TODO: use default port 623
	port, err := tc.IntPort()
	if err != nil {
		return nil, errors.New("port is not a valid number " + tc.Port)
	}

	client, err := ipmi.NewIpmiClient(&ipmi.Connection{
		Hostname:  tc.Host,
		Port:      port,
		Username:  tc.User,
		Password:  tc.Password,
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
	return transports.Capabilities{}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) Client() *ipmi.IpmiClient {
	return t.client
}

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/ipmi/deviceid/", nil
}

func (t *Transport) Device() string {
	return "deviceid"
}
