package gcp

import (
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_GCP {
		return nil, errors.New("backend is not supported for gcp transport")
	}

	if tc.Options == nil || len(tc.Options["project"]) == 0 {
		return nil, errors.New("gcp backend requires a project id. please set option `project`")
	}

	return &Transport{
		projectid: tc.Options["project"],
		opts:      tc.Options,
	}, nil
}

type Transport struct {
	projectid      string
	organizationId string
	opts           map[string]string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("gcp does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("gcp does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Gcp,
	}
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AWS
}
