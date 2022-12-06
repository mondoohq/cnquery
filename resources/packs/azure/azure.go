package azure

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	microsoft_transport "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/resources/packs/azure/info"
	"go.mondoo.com/cnquery/resources/packs/core"
)

var Registry = info.Registry

func init() {
	Init(Registry)
	Registry.Add(core.Registry)
}

func azureTransport(t providers.Instance) (*microsoft_transport.Provider, error) {
	at, ok := t.(*microsoft_transport.Provider)
	if !ok {
		return nil, errors.New("azure resource is not supported on this transport")
	}
	if len(at.SubscriptionID()) == 0 {
		return nil, errors.New("azure resource requires a subscription id")
	}
	return at, nil
}

// TODO: temporary second function to be used only in azuread.* resources. for these, a subscription is not required.
func msGraphTransport(t providers.Instance) (*microsoft_transport.Provider, error) {
	at, ok := t.(*microsoft_transport.Provider)
	if !ok {
		return nil, errors.New("azure resource is not supported on this provider")
	}
	return at, nil
}
