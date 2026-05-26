// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/huggingface/provider"
)

var Config = plugin.Provider{
	Name:            "huggingface",
	ID:              "go.mondoo.com/mql/providers/huggingface",
	Version:         "13.0.1",
	ConnectionTypes: []string{provider.DefaultConnectionType},
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
			},
		},
	},
}
