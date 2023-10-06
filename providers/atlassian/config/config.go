package config

import "go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "atlassian",
	ID:      "go.mondoo.com/cnquery/providers/atlassian",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "atlassian",
			Use:       "atlassian",
			Short:     "Atlassian",
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
