package gcp

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
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
