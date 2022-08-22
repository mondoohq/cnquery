package info

// Load metadata for this resource pack

import (
	_ "embed"

	"go.mondoo.io/mondoo/resources"
)

// fyi this is a workaround for paths: https://github.com/golang/go/issues/46056
//
//go:generate cp ../github.lr.json ./github.lr.json
//go:embed github.lr.json
var info []byte

var Registry = resources.NewRegistry()

func init() {
	if err := Registry.LoadJson(info); err != nil {
		panic(err.Error())
	}
}
