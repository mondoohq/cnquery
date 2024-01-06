// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:       "core",
	ID:         "go.mondoo.com/cnquery/providers/core",
	Version:    "9.1.8",
	Connectors: []plugin.Connector{},
}
