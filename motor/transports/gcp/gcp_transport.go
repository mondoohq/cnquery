package gcp

import (
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

var (
	_ transports.Transport                   = (*Transport)(nil)
	_ transports.TransportPlatformIdentifier = (*Transport)(nil)
)

type ResourceType int

const (
	Unknown ResourceType = iota
	Project
	Organization
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_GCP {
		return nil, errors.New("backend is not supported for gcp transport")
	}

	if tc.Options == nil || (tc.Options["project"] == "" && tc.Options["organization"] == "") {
		return nil, errors.New("gcp backend requires a project id or organization id. please set option `project` or `organization`")
	}

	var resourceType ResourceType
	var id string
	if tc.Options["project"] != "" {
		resourceType = Project
		id = tc.Options["project"]
	} else if tc.Options["organization"] != "" {
		resourceType = Organization
		id = tc.Options["organization"]
	}

	t := &Transport{
		resourceType: resourceType,
		id:           id,
		opts:         tc.Options,
	}

	// verify that we have access to the organization or project
	switch resourceType {
	case Organization:
		_, err := t.GetOrganization(id)
		if err != nil {
			return nil, errors.New("could not find or have no access to organization " + id)
		}
	case Project:
		_, err := t.GetProject(id)
		if err != nil {
			return nil, errors.New("could not find or have no access to project " + id)
		}
	}

	return t, nil
}

type Transport struct {
	resourceType ResourceType
	id           string
	opts         map[string]string
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

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}
