// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/openai/provider"
)

var Config = plugin.Provider{
	Name:            "openai",
	ID:              "go.mondoo.com/mql/providers/openai",
	Version:         "13.0.1",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "openai",
			Use:       "openai",
			Short:     "an OpenAI account",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long: "token",
					Type: plugin.FlagType_String,
					Desc: "OpenAI API key — project key (sk-proj-...) or admin key (sk-admin-...), auto-detected",
				},
				{
					Long: "organization",
					Type: plugin.FlagType_String,
					Desc: "OpenAI organization ID (or set OPENAI_ORG_ID env var)",
				},
				{
					Long: "project",
					Type: plugin.FlagType_String,
					Desc: "OpenAI project ID (or set OPENAI_PROJECT_ID env var)",
				},
				{
					Long: "base-url",
					Type: plugin.FlagType_String,
					Desc: "OpenAI API base URL for custom endpoints (or set OPENAI_BASE_URL env var)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=ai", "provider=openai"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"account": nil,
			},
		},
	},
}
