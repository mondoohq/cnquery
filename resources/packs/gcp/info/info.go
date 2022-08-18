package info

// Load metadata for this resource pack

import (
	_ "embed"

	"go.mondoo.io/mondoo/lumi"
)

// fyi this is a workaround for paths: https://github.com/golang/go/issues/46056
//
//go:generate cp ../gcp.lr.json ./gcp.lr.json
//go:embed gcp.lr.json
var info []byte

var Registry = lumi.NewRegistry()

func init() {
	if err := Registry.LoadJson(info); err != nil {
		panic(err.Error())
	}
}
