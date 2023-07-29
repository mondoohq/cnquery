package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:       "core",
	ID:         "go.mondoo.com/cnquery/providers/core",
	Version:    "9.0.0",
	Connectors: []plugin.Connector{},
}
