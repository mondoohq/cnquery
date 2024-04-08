// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

var Config = plugin.Provider{
	Name:       "core",
	ID:         "go.mondoo.com/cnquery/v9/providers/core",
	Version:    "10.5.0",
	Connectors: []plugin.Connector{},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=other"},
			Key:          "platform",
			Title:        "Other Platform",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
}
