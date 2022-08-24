package k8s

import "go.mondoo.com/cnquery/resources/packs/k8s/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
