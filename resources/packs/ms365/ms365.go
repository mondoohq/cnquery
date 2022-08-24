package ms365

import "go.mondoo.com/cnquery/resources/packs/ms365/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
