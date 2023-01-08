package ms365

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	microsoft "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/resources/packs/ms365/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func microsoftProvider(t providers.Instance) (*microsoft.Provider, error) {
	at, ok := t.(*microsoft.Provider)
	if !ok {
		return nil, errors.New("microsoft resource is not supported on this provider")
	}
	return at, nil
}
