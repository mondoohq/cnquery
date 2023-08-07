package providers

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

const DefaultOsID = "go.mondoo.com/cnquery/providers/os"

var defaultRuntime *Runtime

func DefaultRuntime() *Runtime {
	if defaultRuntime == nil {
		defaultRuntime = Coordinator.NewRuntime()
	}
	return defaultRuntime
}

// DefaultProviders are useful when working in air-gapped environments
// to tell users what providers are used for common connections, when there
// is no other way to find out.
var DefaultProviders Providers = map[string]*Provider{
	"os": {
		Provider: &plugin.Provider{
			Name: "os",
			Connectors: []plugin.Connector{
				{
					Name:  "local",
					Short: "your local system",
				},
				{
					Name:  "ssh",
					Short: "a remote system via SSH",
				},
				{
					Name:  "winrm",
					Short: "a remote system via WinRM",
				},
			},
		},
	},
	"network": {
		Provider: &plugin.Provider{
			Name: "network",
			Connectors: []plugin.Connector{
				{
					Name:  "host",
					Short: "your local system",
				},
			},
		},
	},
}
