// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/gitlab/provider"
)

var Config = plugin.Provider{
	Name:            "gitlab",
	ID:              "go.mondoo.com/cnquery/providers/gitlab",
	Version:         "9.0.1",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "gitlab",
			Use:       "gitlab",
			Short:     "GitLab",
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Provide GitLab personal access token.",
				},
				{
					Long:    "group",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "a GitLab group to scan",
				},
			},
		},
	},
}
