package arista

import (
	"errors"

	"github.com/aristanetworks/goeapi"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(endpoint *transports.TransportConfig) (*Transport, error) {
	node, err := goeapi.Connect("http", "localhost", "admin", "", 8080)
	if err != nil {
		return nil, err
	}

	return &Transport{
		node: node,
	}, nil
}

type Transport struct {
	node *goeapi.Node
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
	return transports.Capabilities{}
}

func (t *Transport) Client() *goeapi.Node {
	return t.node
}
