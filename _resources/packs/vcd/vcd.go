package vcd

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	vcd_provider "go.mondoo.com/cnquery/motor/providers/vcd"
	"go.mondoo.com/cnquery/resources/packs/vcd/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (k *mqlVcd) id() (string, error) {
	return "vcd", nil
}

// vcdProvider returns VCD provider instance to get access to the API
// see https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/landing-extension_typed-queries.html
func vcdProvider(p providers.Instance) (*vcd_provider.Provider, error) {
	at, ok := p.(*vcd_provider.Provider)
	if !ok {
		return nil, errors.New("vcd resource is not supported on this provider")
	}
	return at, nil
}
