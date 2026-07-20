// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/huggingface/connection"
	"go.mondoo.com/mql/v13/providers/huggingface/provider"
)

var Config = plugin.Provider{
	Name:            "huggingface",
	ID:              "go.mondoo.com/mql/providers/huggingface",
	Version:         "13.1.9",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:      "huggingface",
			Use:       "huggingface",
			Short:     "Hugging Face",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long: "token",
					Type: plugin.FlagType_String,
					Desc: "Hugging Face API token (or set HF_TOKEN env var)",
				},
				{
					Long: "namespace",
					Type: plugin.FlagType_String,
					Desc: "Target a specific user or organization namespace (e.g. openai)",
				},
				{
					Long:    "namespace-type",
					Type:    plugin.FlagType_String,
					Default: "user",
					Desc:    "Type of the namespace: user or org",
				},
			},
		},
	},
}
