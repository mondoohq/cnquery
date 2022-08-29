package info

// Load metadata for this resource pack

import (
	_ "embed"
	"encoding/json"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/lr/docs"
)

//go:embed vsphere.lr.json
var info []byte

//go:embed vsphere.lr.manifest.json
var manifest []byte

// Registry contains the resource info necessary for the compiler to work with this pack.
var Registry = resources.NewRegistry()

// ResourceDocs contains additional resource metadata for the compiler to use.
var ResourceDocs docs.LrDocs

func init() {
	if err := Registry.LoadJson(info); err != nil {
		panic(err.Error())
	}

	if err := json.Unmarshal(manifest, &ResourceDocs); err != nil {
		panic(err.Error())
	}
}
