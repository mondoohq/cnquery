package google

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers/os"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/os/fsutil"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

type ResourceType int

const (
	Unknown ResourceType = iota
	Project
	Organization
)

func New(pCfg *providers.Config) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_GCP {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	if pCfg.Options == nil || (pCfg.Options["project-id"] == "" && pCfg.Options["project"] == "" && pCfg.Options["organization-id"] == "" && pCfg.Options["organization"] == "") {
		return nil, errors.New("gcp provider requires a project id or organization id. please set option `project` or `organization`")
	}

	var resourceType ResourceType
	var id string
	if pCfg.Options["project-id"] != "" {
		resourceType = Project
		id = pCfg.Options["project-id"]
	} else if pCfg.Options["project"] != "" {
		// deprecated, use project-id
		resourceType = Project
		id = pCfg.Options["project"]
	} else if pCfg.Options["organization-id"] != "" {
		resourceType = Organization
		id = pCfg.Options["organization-id"]
	} else if pCfg.Options["organization"] != "" {
		resourceType = Organization
		id = pCfg.Options["organization"]
	}

	t := &Provider{
		resourceType: resourceType,
		id:           id,
		opts:         pCfg.Options,
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

type Provider struct {
	resourceType ResourceType
	id           string
	opts         map[string]string
}

func (p *Provider) RunCommand(command string) (*os.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FileInfo(path string) (os.FileInfoDetails, error) {
	return os.FileInfoDetails{}, providers.ErrFileInfoNotImplemented
}

func (p *Provider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Gcp,
	}
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_GCP
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
