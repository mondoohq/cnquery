package providers

// Uncomment any provider you want to load directly into the binary.
// This is primarily useful for debugging purposes, if you want to
// trace into any provider without having to debug the plugin
// connection separately.

import (
	// osconf "go.mondoo.com/cnquery/providers/os/config"
	// os "go.mondoo.com/cnquery/providers/os/provider"
	"go.mondoo.com/cnquery/providers/plugin"
)

type builtinProvider struct {
	Runtime *ProviderRuntime
	Config  *plugin.Provider
}

var builtinProviders = map[string]*builtinProvider{
	// "os": {
	// 	Runtime: &ProviderRuntime{
	// 		Name:     "os",
	// 		Plugin:   &os.Service{},
	// 		isClosed: false,
	// 	},
	// 	Config: &osconf.Config,
	// },
}
