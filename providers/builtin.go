package providers

// Uncomment any provider you want to load directly into the binary.
// This is primarily useful for debugging purposes, if you want to
// trace into any provider without having to debug the plugin
// connection separately.

import (
	_ "embed"
	"encoding/json"
	osfs "os"

	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
	coreconf "go.mondoo.com/cnquery/providers/core/config"
	core "go.mondoo.com/cnquery/providers/core/provider"
	// osconf "go.mondoo.com/cnquery/providers/os/config"
	// os "go.mondoo.com/cnquery/providers/os/provider"
)

var BuiltinCoreID = coreconf.Config.ID

//go:embed core/resources/core.resources.json
var coreInfo []byte

// //go:embed os/resources/os.resources.json
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

func MustLoadSchemaFromFile(name string, path string) *resources.Schema {
	raw, err := osfs.ReadFile(path)
	if err != nil {
		panic("cannot read schema file: " + path)
	}
	return MustLoadSchema(name, raw)
}
