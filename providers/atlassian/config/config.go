// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/atlassian/provider"
)

var Config = plugin.Provider{
	Name:    "atlassian",
	ID:      "go.mondoo.com/cnquery/providers/atlassian",
	Version: "9.0.0",
	ConnectionTypes: []string{
		provider.DefaultConnectionType,
		"jira",
		"admin",
		"confluence",
		"scim",
	},
	Connectors: []plugin.Connector{
		{
			Name:      "atlassian",
			Use:       "atlassian",
			Short:     "Atlassian Cloud",
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "admin-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide Atlassian admin api token (used for Atlassian admin).",
				},
				{
					Long:    "host",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide Atlassian hostname (e.g. https://example.atlassian.net).",
				},
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide Atlassian user name (e.g. example@example.com).",
				},
				{
					Long:    "user-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide Atlassian user api token (used for Jira / Confluence).",
				},
				{
					Long:    "scim-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide Atlassian scim api token (used for scim).",
				},
			},
		},
	},
}
