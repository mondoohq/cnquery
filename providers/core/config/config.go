package config

import "go.mondoo.com/cnquery/providers/plugin"

var Config = plugin.Provider{
	Name:       "core",
	ID:         "go.mondoo.com/cnquery/providers/core",
	Connectors: []plugin.Connector{},
}
