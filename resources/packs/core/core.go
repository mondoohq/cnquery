package core

import "go.mondoo.io/mondoo/resources/packs/core/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
