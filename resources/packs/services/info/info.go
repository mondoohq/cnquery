package info

// Load metadata for this resource pack

import (
	_ "embed"

	"go.mondoo.io/mondoo/resources"
)

// fyi this is a workaround for paths: https://github.com/golang/go/issues/46056
//
//go:generate cp ../services.lr.json ./services.lr.json
//go:embed services.lr.json
var info []byte

var Registry = resources.NewRegistry()

func init() {
	if err := Registry.LoadJson(info); err != nil {
		panic(err.Error())
	}
}
