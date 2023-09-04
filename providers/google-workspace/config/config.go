// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/google-workspace/provider"
)

var Config = plugin.Provider{
	Name:            "google-workspace",
	ID:              "go.mondoo.com/cnquery/providers/google-workspace",
	Version:         "9.0.0",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "google-workspace",
			Use:       "google-workspace",
			Short:     "Google Workspace",
			Aliases:   []string{"googleworkspace"},
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "credentials-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "The path to the service account credentials to access the APIs with",
				},
				{
					Long:    "customer-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Specify the Google Workspace customer id to scan",
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
