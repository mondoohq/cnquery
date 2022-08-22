package ms365

import "go.mondoo.io/mondoo/resources/packs/ms365/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
