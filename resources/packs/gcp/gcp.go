package gcp

import "go.mondoo.com/cnquery/resources/packs/gcp/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
