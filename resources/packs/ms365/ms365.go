package ms365

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	ms365_provider "go.mondoo.com/cnquery/motor/providers/microsoft/ms365"
	"go.mondoo.com/cnquery/resources/packs/ms365/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func ms365Provider(t providers.Instance) (*ms365_provider.Provider, error) {
	at, ok := t.(*ms365_provider.Provider)
	if !ok {
		return nil, errors.New("ms365 resource is not supported on this provider")
	}
	return at, nil
}
