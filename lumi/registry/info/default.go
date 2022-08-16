package info

/*

This file contains the metadata for Lumi's default resource registry.
No implementation code is loaded. It is also a prerequisite to a fully
functioning registry.

*/

import (
	_ "embed"
	"encoding/json"
	"sort"

	"go.mondoo.io/mondoo/lumi"
)

// fyi this is a workaround for paths: https://github.com/golang/go/issues/46056
//
//go:generate cp ../../resources/core.lr.json ./core.lr.json
//go:embed core.lr.json
var coreInfo []byte

var Default = lumi.NewRegistry()

func init() {
	schema := lumi.Schema{}
	if err := json.Unmarshal(coreInfo, &schema); err != nil {
		panic("cannot load embedded core resource schema")
	}

	// since we establish the resource chain of any missing resources,
	// it is important to add things in the right order (for now)
	keys := make([]string, len(schema.Resources))
	var i int
	for k := range schema.Resources {
		keys[i] = k
		i++
	}

	sort.Strings(keys)
	for i := range keys {
		if err := Default.AddResourceInfo(schema.Resources[keys[i]]); err != nil {
			panic("failed to add resource info: " + err.Error())
		}
	}
}
