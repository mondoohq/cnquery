package azure

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	azure_transport "go.mondoo.com/cnquery/motor/providers/azure"
	"go.mondoo.com/cnquery/resources/packs/azure/info"
	"go.mondoo.com/cnquery/resources/packs/core"
)

var Registry = info.Registry

func init() {
	Init(Registry)
	Registry.Add(core.Registry)
}

func azuretransport(t providers.Instance) (*azure_transport.Provider, error) {
	at, ok := t.(*azure_transport.Provider)
	if !ok {
		return nil, errors.New("azure resource is not supported on this transport")
	}
	return at, nil
}
