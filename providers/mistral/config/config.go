// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/mistral/connection"
	"go.mondoo.com/mql/v13/providers/mistral/provider"
)

var Config = plugin.Provider{
	Name:            "mistral",
	ID:              "go.mondoo.com/mql/providers/mistral",
	Version:         "13.0.9",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "mistral",
			Use:   "mistral",
			Short: "a Mistral AI workspace",
			Long: `Use the mistral provider to query models, fine-tuning jobs, uploaded
files, and batch inference jobs in a Mistral AI workspace.

Important: Mistral AI API keys are scoped to a single workspace. To scan
all workspaces in an organization, add each workspace as a separate asset
with its own API key.

Examples:
  mql shell mistral --token <api-key>
  mql shell mistral --token <api-key> --workspace my-workspace
  mql shell mistral   # uses MISTRAL_API_KEY or MISTRAL_KEY env var

Notes:
  The provider queries the Mistral AI REST API at https://api.mistral.ai/v1.
  An API key is required; generate one at https://console.mistral.ai/api-keys.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionToken,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Mistral AI API key (or set MISTRAL_API_KEY env var)",
				},
				{
					Long:    connection.OptionWorkspace,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Workspace name for asset identification (API keys are workspace-scoped)",
				},
				{
					Long:    connection.OptionBaseURL,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "API base URL (default: https://api.mistral.ai)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=ai", "provider=mistral"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"workspace": nil,
			},
		},
	},
}
