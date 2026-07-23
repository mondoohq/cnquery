// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/providers/snowflake/provider"
)

var Config = plugin.Provider{
	Name:            "snowflake",
	ID:              "go.mondoo.com/mql/v13/providers/snowflake",
	Version:         "13.4.2",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "snowflake",
			Use:   "snowflake",
			Short: "a Snowflake account",
			Long: `Use the snowflake provider to query a Snowflake account.

To access a Snowflake account, you must first authenticate with Snowflake. The recommended method is a programmatic access token (PAT), which replaces password sign-ins that Snowflake is phasing out. Generate a PAT for your user in Snowsight and pass it with --token. Key-pair authentication (--identity-file) is also supported; to learn how, read https://docs.snowflake.com/en/user-guide/key-pair-auth.

Once you successfully authenticate, you can scan or query the Snowflake account.

Example:
  cnspec shell snowflake --account <account id> --region <region> --user <your id> --role <the role you use> --token <your PAT>
  cnspec scan snowflake --account <account id> --region <region> --user <your id> --role <the role you use> --token <your PAT>
`,
			Discovery: []string{
				connection.DiscoveryAuto,
				connection.DiscoveryAll,
				connection.DiscoveryDatabases,
				connection.DiscoveryNone,
			},
			Flags: []plugin.Flag{
				{
					Long:    "user",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Snowflake user name",
				},
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for the connection password",
					ConfigEntry: "-",
				},
				{
					Long:        "token",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the programmatic access token (PAT)",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:    "identity-file",
					Short:   "i",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Select a file from which to read the identity (private key) for public key authentication",
				},
				{
					Long:    "account",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Snowflake account ID",
				},
				{
					Long:    "region",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Snowflake region",
				},
				{
					Long:    "role",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The role you use to access Snowflake",
				},
			},
		},
	},
}
