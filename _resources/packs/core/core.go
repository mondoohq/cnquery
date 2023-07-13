package core

import "go.mondoo.com/cnquery/resources/packs/core/info"

const MissingUpstreamErr = `To use this resource, you must authenticate with Mondoo Platform.
To learn how, read: 
https://mondoo.com/docs/cnspec/cnspec-adv-install/registration/`

var Registry = info.Registry

func init() {
	Init(Registry)
}
