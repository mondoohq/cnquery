package config

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/virustotal/connection"
)

var Config = plugin.Provider{
	Name:            "virustotal",
	ID:              "go.mondoo.com/cnquery/v12/providers/virustotal",
	Version:         "0.0.1",
	ConnectionTypes: []string{"virustotal"},
	Connectors: []plugin.Connector{
		{
			Name:  "virustotal",
			Use:   "virustotal",
			Short: "a VirusTotal API account",
			Long: `Use the virustotal provider to explore enrichment data from the VirusTotal API.

If you set the VT_API_KEY or VIRUSTOTAL_API_KEY environment variables, you can omit the --api-key flag.

Examples:
  cnquery shell virustotal --api-key <api-key>
`,
			MinArgs: 0,
			MaxArgs: 0,
			Discovery: []string{
				connection.DiscoveryNone,
			},
			Flags: []plugin.Flag{
				{
					Long:    "api-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "VirusTotal API key",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=threat-intel", "category=virustotal"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"profile": nil,
				"domain":  nil,
				"ip":      nil,
				"hash":    nil,
			},
		},
	},
}
