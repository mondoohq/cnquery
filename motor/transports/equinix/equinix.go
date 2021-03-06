package equinix

import (
	"github.com/cockroachdb/errors"
	"github.com/packethost/packngo"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_EQUINIX_METAL {
		return nil, errors.New("backend is not supported for equinix transport")
	}

	projectId := tc.Options["projectID"]

	if tc.Options == nil || len(projectId) == 0 {
		return nil, errors.New("equinix backend requires an project id")
	}

	c, err := packngo.NewClient()
	if err != nil {
		return nil, err
	}

	// NOTE: we cannot check the project itself because it throws a 404
	// https://github.com/packethost/packngo/issues/245
	//project, _, err := c.Projects.Get(projectId, nil)
	//if err != nil {
	//	return nil, errors.Wrap(err, "could not find the requested equinix project: "+projectId)
	//}

	ps, _, err := c.Projects.List(nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve equinix projects")
	}

	var project *packngo.Project
	for _, p := range ps {
		if p.ID == projectId {
			project = &p
		}
	}
	if project == nil {
		return nil, errors.Wrap(err, "could not find the requested equinix project: "+projectId)
	}

	return &Transport{
		client:    c,
		projectId: projectId,
		project:   project,
	}, nil
}

type Transport struct {
	client    *packngo.Client
	projectId string
	project   *packngo.Project
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("equinix does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("equinix does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Equinix,
	}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_EQUINIX_METAL
}

func (t *Transport) Client() *packngo.Client {
	return t.client
}

func (t *Transport) Project() *packngo.Project {
	return t.project
}
