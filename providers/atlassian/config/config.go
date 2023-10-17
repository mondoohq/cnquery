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
			Name:  "atlassian",
			Use:   "atlassian",
			Short: "Atlassian",
			Long: `atlassian is designed for querying resources within Atlassian Cloud.

The provider offers 4 subcommands:
1. Admin
"cnquery shell atlassian admin" requires either the env variable "ATLASSIAN_ADMIN_TOKEN" or the --admin-token flag to be set

2. Jira
"cnquery shell atlassian jira" requires either the env variables "ATLASSIAN_USER", "ATLASSIAN_HOST" and "ATLASSIAN_USER_TOKEN" or 
the flags --user, --host, --user-token to be set

3. Confluence
"cnquery shell atlassian confluence" requires either the env variables "ATLASSIAN_USER", "ATLASSIAN_HOST" and "ATLASSIAN_USER_TOKEN" or 
the flags --user, --host, --user-token to be set

4. SCIM
"cnquery shell atlassian scim DIRECTORYID" requires a directory-id and either the env variable "ATLASSIAN_SCIM_TOKEN" or the --scim-token flag to be set
You receiv both the token and the directory-id from atlassian when you setup an identity provider.
`,
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
