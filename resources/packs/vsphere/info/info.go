package info

// Load metadata for this resource pack

import (
	_ "embed"

	"go.mondoo.com/cnquery/resources"
)

// fyi this is a workaround for paths: https://github.com/golang/go/issues/46056
//
//go:generate cp ../vsphere.lr.json ./vsphere.lr.json
//go:embed vsphere.lr.json
var info []byte

var Registry = resources.NewRegistry()

func init() {
	if err := Registry.LoadJson(info); err != nil {
		panic(err.Error())
	}
}
