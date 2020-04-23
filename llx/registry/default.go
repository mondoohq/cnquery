package registry

// These functions are needed to be located here to avoid dependency cycles
// since resources --depends--> lumi
// and this runtime --depends--> lumi + resources + motor

import (
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources"
)

// Default with core resources
var Default = lumi.NewRegistry()

func init() {
	resources.Init(Default)
}
