// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ollama/provider"
)

var Config = plugin.Provider{
	Name:            "ollama",
	ID:              "go.mondoo.com/mql/providers/ollama",
	Version:         "13.1.2",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       provider.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "ollama",
			Use:   "ollama",
			Short: "an Ollama instance",
			Long: `Use the ollama provider to query the models and configuration of an Ollama instance.

Examples:
  cnspec shell ollama
  cnspec shell ollama --host http://<HOST>:11434
  cnspec scan ollama --token <API-TOKEN>

Notes:
  Without the host flag, Ollama is queried at http://localhost:11434. You can also set
  the OLLAMA_HOST environment variable. Use the token flag, or OLLAMA_API_TOKEN, to
  authenticate against a cloud-hosted Ollama instance.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long: "host",
					Type: plugin.FlagType_String,
					Desc: "Ollama host address (or set OLLAMA_HOST env var, default: http://localhost:11434)",
				},
				{
					Long: "token",
					Type: plugin.FlagType_String,
					Desc: "Ollama API token for cloud authentication (or set OLLAMA_API_TOKEN env var)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=ai", "provider=ollama"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"instance": nil,
			},
		},
	},
}
