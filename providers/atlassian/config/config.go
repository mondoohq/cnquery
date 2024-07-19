// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/atlassian/provider"
)

var Config = plugin.Provider{
	Name:    "atlassian",
	ID:      "go.mondoo.com/cnquery/v9/providers/atlassian",
	Version: "11.0.23",
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
			Short: "an Atlassian Cloud Jira, Confluence or Bitbucket instance",
			Long: `atlassian is designed for querying resources within Atlassian Cloud, including Jira, Confluence, and SCIM.

Available Commands:
  admin                     Specifies the Atlassian admin to interact with.
  jira                      Specifies the Jira instance to interact with.
  confluence                Specifies the Confluence instance to interact with.
  scim                      Specifies the SCIM instance to interact with.

Examples:
  cnquery shell atlassian admin --admin-token <token>
  cnquery shell atlassian jira --host <host> --user <user> --user-token <token>
  cnquery shell atlassian confluence --host <host> --user <user> --user-token <token>
  cnquery shell atlassian scim <directory-id> --scim-token <token>

If the ATLASSIAN_ADMIN_TOKEN environment variable is set, the admin-token flag is not required. If the ATLASSIAN_USER,
ATLASSIAN_HOST, and ATLASSIAN_USER_TOKEN environment variables are set, the user, host, and user-token flags are not required.

For SCIM, you receive both the token and the directory-id from Atlassian when you setup an identity provider.
`,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "admin-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Atlassian admin API token (used for Atlassian admin)",
				},
				{
					Long:    "host",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Atlassian hostname (e.g. https://example.atlassian.net)",
				},
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Atlassian user name (e.g. example@example.com)",
				},
				{
					Long:    "user-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Atlassian user API token (used for Jira / Confluence)",
				},
				{
					Long:    "scim-token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Atlassian SCIM API token (used for SCIM)",
				},
			},
		},
	},
}
