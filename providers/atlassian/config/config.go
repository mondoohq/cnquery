package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/atlassian/provider"
)

var Config = plugin.Provider{
	Name:            "atlassian",
	ID:              "go.mondoo.com/cnquery/providers/atlassian",
	Version:         "9.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
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
