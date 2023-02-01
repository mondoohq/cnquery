package arista

import "go.mondoo.com/cnquery/resources/packs/arista/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
