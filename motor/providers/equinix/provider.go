package equinix

import (
	"errors"
	"github.com/packethost/packngo"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(cfg *providers.Config) (*Provider, error) {
	if cfg.Backend != providers.ProviderType_EQUINIX_METAL {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	projectId := cfg.Options["projectID"]

	if cfg.Options == nil || len(projectId) == 0 {
		return nil, errors.New("equinix provider requires an project id")
	}

	c, err := packngo.NewClient()
	if err != nil {
		return nil, err
	}

	// NOTE: we cannot check the project itself because it throws a 404
	// https://github.com/packethost/packngo/issues/245
	//project, _, err := c.Projects.Get(projectId, nil)
	//if err != nil {
	//	return nil, errors.Join(err, errors.New("could not find the requested equinix project: "+projectId))
	//}

	ps, _, err := c.Projects.List(nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("cannot retrieve equinix projects"))
	}

	var project *packngo.Project
	for _, p := range ps {
		if p.ID == projectId {
			project = &p
		}
	}
	if project == nil {
		return nil, errors.Join(err, errors.New("could not find the requested equinix project: "+projectId))
	}

	return &Provider{
		client:    c,
		projectId: projectId,
		project:   project,
	}, nil
}

type Provider struct {
	client    *packngo.Client
	projectId string
	project   *packngo.Project
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Equinix,
	}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_EQUINIX_METAL
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Client() *packngo.Client {
	return p.client
}

func (p *Provider) Project() *packngo.Project {
	return p.project
}
