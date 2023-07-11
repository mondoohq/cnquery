package providers

// Uncomment any provider you want to load directly into the binary.
// This is primarily useful for debugging purposes, if you want to
// trace into any provider without having to debug the plugin
// connection separately.

import (
	_ "embed"
	"encoding/json"

	coreconf "go.mondoo.com/cnquery/providers/core/config"
	core "go.mondoo.com/cnquery/providers/core/provider"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/resources"
	// osconf "go.mondoo.com/cnquery/providers/os/config"
	// os "go.mondoo.com/cnquery/providers/os/provider"
)

var BuiltinCoreID = coreconf.Config.ID

const DefaultOsID = "go.mondoo.com/cnquery/providers/os"

//go:embed core/dist/core.resources.json
var coreInfo []byte

// //go:embed os/dist/os.resources.json
// var osInfo []byte

var builtinProviders = map[string]*builtinProvider{
	coreconf.Config.ID: {
		Runtime: &RunningProvider{
			Name:     coreconf.Config.Name,
			ID:       coreconf.Config.ID,
			Plugin:   core.Init(),
			Schema:   MustLoadSchema("core", coreInfo),
			isClosed: false,
		},
		Config: &coreconf.Config,
	},
	// osconf.Config.ID: {
	// 	Runtime: &RunningProvider{
	// 		Name:     osconf.Config.Name,
	// 		ID:       osconf.Config.ID,
	// 		Plugin:   os.Init(),
	// 		Schema:   MustLoadSchema("os", osInfo),
	// 		isClosed: false,
	// 	},
	// 	Config: &osconf.Config,
	// },
}

type builtinProvider struct {
	Runtime *RunningProvider
	Config  *plugin.Provider
}

func MustLoadSchema(name string, data []byte) *resources.Schema {
	var res resources.Schema
	if err := json.Unmarshal(data, &res); err != nil {
		panic("failed to embed schema for " + name)
	}
	return &res
}
