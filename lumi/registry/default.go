package registry

// These functions are needed to be located here to avoid dependency cycles
// since resources --depends--> lumi
// and this runtime --depends--> lumi + resources + motor

import (
	"go.mondoo.io/mondoo/lumi/registry/info"
	"go.mondoo.io/mondoo/lumi/resources"
)

// we import this from Info to fill in all the metadata first
var Default = info.Default

func init() {
	resources.Init(Default)
}
