// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/google-workspace/provider"
)

var Config = plugin.Provider{
	Name:            "google-workspace",
	ID:              "go.mondoo.com/cnquery/v9/providers/google-workspace",
	Version:         "11.0.4",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "google-workspace",
			Use:   "google-workspace [--credentials-path <credentials-path>] [--customer-id <customer-id>] [--impersonated-user-email <impersonated-user-email>]",
			Short: "a Google Workspace account",
			Long: `google-workspace is designed for querying resources within for a Google Workspace domain.

The provider requires three flags to be set:
'--customer-id <customer-id>': This flag is used to specify the unique ID of the Google Workspace customer. 
The customer ID is an immutable, unique identifier for a Google Workspace account.

'--impersonated-user-email <user-email>': This flag is used to specify the email address of the user to 
impersonate in this session. This is useful when the user executing the command does not have the necessary 
permissions, but can impersonate a user who does.

'--credentials-path <credentials-file-path>': This flag is used to specify the file path to the credentials 
file (typically a JSON file) that should be used for authentication.

In case GOOGLE_APPLICATION_CREDENTIALS environment variable is set, the --credentials-path flag can be omitted.
`,

			Aliases:   []string{"googleworkspace"},
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "credentials-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The path to the service account credentials to access the APIs with",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "customer-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Specify the Google Workspace customer id to scan",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "impersonated-user-email",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The impersonated user's email with access to the Admin APIs",
				},
			},
		},
	},
}
