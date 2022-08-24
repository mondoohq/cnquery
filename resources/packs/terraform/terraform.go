package terraform

import "go.mondoo.com/cnquery/resources/packs/terraform/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
