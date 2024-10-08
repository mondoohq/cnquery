// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/gitlab/provider"
)

var Config = plugin.Provider{
	Name:    "gitlab",
	ID:      "go.mondoo.com/cnquery/v9/providers/gitlab",
	Version: "11.1.27",
	ConnectionTypes: []string{
		provider.ConnectionType,
		provider.GitlabGroupConnection,
		provider.GitlabProjectConnection,
	},
	Connectors: []plugin.Connector{
		{
			Name:  "gitlab",
			Use:   "gitlab",
			Short: "a GitLab group or project",
			Discovery: []string{
				provider.DiscoveryAuto,
				provider.DiscoveryGroup,
				provider.DiscoveryProject,
				provider.DiscoveryTerraform,
			},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitLab personal access token",
				},
				{
					Long:    "group",
					Type:    plugin.FlagType_String,
					Option:  plugin.FlagOption_Required,
					Default: "",
					Desc:    "GitLab group to scan",
				},
				{
					Long:    "project",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitLab project to scan",
				},
				{
					Long:    "url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Custom GitLab base url",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=gitlab"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"project": nil,
				"group":   nil,
			},
		},
	},
}
