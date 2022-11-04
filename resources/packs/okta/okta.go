package okta

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	okta_provider "go.mondoo.com/cnquery/motor/providers/okta"
	"go.mondoo.com/cnquery/resources/packs/okta/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (k *mqlOkta) id() (string, error) {
	return "okta", nil
}

func oktaProvider(p providers.Instance) (*okta_provider.Provider, error) {
	at, ok := p.(*okta_provider.Provider)
	if !ok {
		return nil, errors.New("okta resource is not supported on this provider")
	}
	return at, nil
}

const queryLimit = 200
