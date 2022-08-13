package resources

import (
	"errors"

	"go.mondoo.io/mondoo/motor/providers"
	azure_transport "go.mondoo.io/mondoo/motor/providers/azure"
)

func azuretransport(t providers.Transport) (*azure_transport.Provider, error) {
	at, ok := t.(*azure_transport.Provider)
	if !ok {
		return nil, errors.New("azure resource is not supported on this transport")
	}
	return at, nil
}
