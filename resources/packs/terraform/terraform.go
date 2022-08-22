package terraform

import "go.mondoo.io/mondoo/resources/packs/terraform/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
