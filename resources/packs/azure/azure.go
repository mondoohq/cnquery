package azure

import (
	"errors"

	"go.mondoo.io/mondoo/motor/providers"
	azure_transport "go.mondoo.io/mondoo/motor/providers/azure"
	"go.mondoo.io/mondoo/resources/packs/azure/info"
	"go.mondoo.io/mondoo/resources/packs/core"
)

var Registry = info.Registry

func init() {
	Init(Registry)
	Registry.Add(core.Registry)
}

func azuretransport(t providers.Transport) (*azure_transport.Provider, error) {
	at, ok := t.(*azure_transport.Provider)
	if !ok {
		return nil, errors.New("azure resource is not supported on this transport")
	}
	return at, nil
}
