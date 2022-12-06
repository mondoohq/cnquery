package gcp

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources/packs/gcp/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func gcpProvider(t providers.Instance) (*gcp_provider.Provider, error) {
	provider, ok := t.(*gcp_provider.Provider)
	if !ok {
		return nil, errors.New("gcp resource is not supported on this transport")
	}
	return provider, nil
}

type zone struct {
	ProjectID string
	Name      string
}

func parseZone(zoneResourceName string) (*zone, error) {
	// we can reuse the mrn parser since google resource names are very similar
	res, err := mrn.NewMRN(zoneResourceName)
	if err != nil {
		return nil, err
	}

	name, err := res.ResourceID("zones")
	if err != nil {
		return nil, err
	}

	projectId, err := res.ResourceID("projects")
	if err != nil {
		return nil, err
	}

	return &zone{
		Name:      name,
		ProjectID: projectId,
	}, nil
}
