package ipmi

import (
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

// New creates a winrm client and establishes a connection to verify the connection
func New(endpoint *transports.TransportConfig) (*Transport, error) {
	return &Transport{}, nil
}

type Transport struct{}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("ipmi does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("ipmi does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/ipmi/deviceid/", nil
}

func (t *Transport) Device() string {
	return "deviceid"
}
