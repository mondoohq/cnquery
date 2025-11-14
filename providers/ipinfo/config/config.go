package config

import (
  "go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
  "go.mondoo.com/cnquery/v12/providers/ipinfo/provider"
)

var Config = plugin.Provider{
	Name:    "ipinfo",
	ID:      "go.mondoo.com/cnquery/v12/providers/ipinfo",
	Version: "10.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "ipinfo",
			Use:       "ipinfo",
			Short:     "ipinfo",
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
