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
			Short:     "atlassian",
			MaxArgs:   2,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
