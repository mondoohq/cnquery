package services

import "go.mondoo.io/mondoo/resources/packs/services/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
