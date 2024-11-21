// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/provider"
)

var Config = plugin.Provider{
	Name:            "snowflake",
	ID:              "go.mondoo.com/cnquery/v11/providers/snowflake",
	Version:         "11.0.28",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "snowflake",
			Use:   "snowflake",
			Short: "a Snowflake account",
			Long: `Use the snowflake provider to query a Snowflake account.

To access a Snowflake account, you must first authenticate with Snowflake. To do so, create an RSA key pair and assign the public key to your user account using Snowsight. To learn how, read https://docs.snowflake.com/en/user-guide/key-pair-auth. Then, in your shell, run:

shell snowflake --account <account id> --region <region> --user <your id>  --role <the role you use> --private-key <path to your private RSA key>

Once you successfully authenticate, you can scan or query the Snowflake account.

Examples:
  cnquery shell snowflake --account <account id> --region <region> --user <your id>  --role <the role you use> --private-key <path to your private RSA key>
  cnspec scan snowflake --account <account id> --region <region> --user <your id>  --role <the role you use> --private-key <path to your private RSA key>
`,
			Discovery: []string{},
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
