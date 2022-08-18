package gcp

import "go.mondoo.io/mondoo/resources/packs/gcp/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
