package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "aws",
	ID:      "go.mondoo.com/cnquery/providers/aws",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "aws",
			Use:       "aws",
			Short:     "aws account",
			MinArgs:   0,
			MaxArgs:   0,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
