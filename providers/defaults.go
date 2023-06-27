package providers

import "go.mondoo.com/cnquery/providers/plugin"

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
}
