package providers

// Uncomment any provider you want to load directly into the binary.
// This is primarily useful for debugging purposes, if you want to
// trace into any provider without having to debug the plugin
// connection separately.

import (
	_ "embed"
	"encoding/json"

	// osconf "go.mondoo.com/cnquery/providers/os/config"
	// os "go.mondoo.com/cnquery/providers/os/provider"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/resources"
)

// //go:embed os/dist/os.resources.json
// var osInfo []byte

var builtinProviders = map[string]*builtinProvider{
	// "os": {
	// 	Runtime: &RunningProvider{
	// 		Name:     "os",
	// 		Plugin:   os.Init(),
	// 		Schema:   mustLoadSchema("os", osInfo),
	// 		isClosed: false,
	// 	},
	// 	Config: &osconf.Config,
	// },
}

type builtinProvider struct {
	Runtime *RunningProvider
	Config  *plugin.Provider
}

func mustLoadSchema(name string, data []byte) *resources.Schema {
	var res resources.Schema
	if err := json.Unmarshal(data, &res); err != nil {
		panic("failed to embed schema for " + name)
	}
	return &res
}
