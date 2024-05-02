// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/github/provider"
)

var Config = plugin.Provider{
	Name:            "github",
	ID:              "go.mondoo.com/cnquery/v9/providers/github",
	Version:         "11.1.2",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "github",
			Use:       "github",
			Short:     "a GitHub organization or repository",
			MinArgs:   2,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide GitHub personal access token.",
				},
				{
					Long:    "repos-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Filter out repositories matching these names.",
				},
				{
					Long:    "repos",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Only include repositories with matching names.",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=github"},
			Key:          "organization",
			Title:        "Organization",
			Values: map[string]*inventory.AssetUrlBranch{
				"organization": {
					Key:   "organization",
					Title: "Organization",
					Values: map[string]*inventory.AssetUrlBranch{
						"organization": nil,
						"*": {
							Key:   "repository",
							Title: "Repository",
							Values: map[string]*inventory.AssetUrlBranch{
								"*": nil,
							},
						},
					},
				},
				"user": {
					Key:   "user",
					Title: "User",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": nil,
					},
				},
			},
		},
	},
}
