// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/google-workspace/provider"
)

var Config = plugin.Provider{
	Name:            "google-workspace",
	ID:              "go.mondoo.com/cnquery/v9/providers/google-workspace",
	Version:         "11.1.12",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "google-workspace",
			Use:   "google-workspace [--credentials-path <credentials-path>] [--customer-id <customer-id>] [--impersonated-user-email <impersonated-user-email>]",
			Short: "a Google Workspace account",
			Long: `Use the google-workspace provider to query resources in a Google Workspace domain.

The provider requires these three flags:

* '--customer-id <customer-id>': This flag specifies the unique ID of the Google Workspace customer. The customer ID is an immutable, unique identifier for a Google Workspace account.

* '--impersonated-user-email <user-email>': This flag specifies the email address of the user to impersonate in the session. This is useful when the user executing the command does not have the necessary permissions, but can impersonate a user who does.

* '--credentials-path <credentials-file-path>': This flag specifies the path to the credentials file (typically a JSON file) to use for authentication. You can omit this flag if the GOOGLE_APPLICATION_CREDENTIALS environment variable is set.
`,

			Aliases:   []string{"googleworkspace"},
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "credentials-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to the service account credentials with which to access the APIs",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "customer-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Google Workspace customer ID to scan",
					Option:  plugin.FlagOption_Required,
				},
				{
					Long:    "impersonated-user-email",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Email address of the user to impersonate in the session",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=google-workspace"},
			Key:          "customer",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
}
