package core

import "go.mondoo.com/cnquery/resources/packs/core/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
