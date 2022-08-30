package terraform

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/terraform"
	"go.mondoo.com/cnquery/resources/packs/terraform/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func terraformProvider(t providers.Instance) (*terraform.Provider, error) {
	gt, ok := t.(*terraform.Provider)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}
