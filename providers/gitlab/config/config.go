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
	Version: "11.1.44",
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
			Long: `Use the gitlab provider to query resources within GitLab groups and projects.

			Available commands:
				group					              GitLab group
				project                     GitLab project
			
			Examples:
				cnspec scan gitlab --group <GROUP_NAME> --token <YOUR_TOKEN>
				cnspec scan gitlab --discover projects --token <YOUR_TOKEN>
				cnspec scan gitlab --group <GROUP_NAME> --project <PROJECT_NAME> --token <YOUR_TOKEN>
				cnspec scan gitlab --group <GROUP_NAME> --discover projects --token <YOUR_TOKEN>
				cnspec scan gitlab --discover terraform --token <YOUR_TOKEN>
				cnquery shell gitlab --group <GROUP_NAME> --token <YOUR_TOKEN>
				cnquery shell gitlab --group <GROUP_NAME> --project <PROJECT_NAME> --token <YOUR_TOKEN>
			
			Notes:
				Mondoo needs a personal access token to scan a GitHub group or project. The token's level of access determines how much information Mondoo can retrieve. Instead of providing a token with every command, you can supply your personal access token to Mondoo by setting the GITLAB_TOKEN environment variable. To learn how, read https://mondoo.com/docs/cnspec/saas/gitlab/.
			`,
			Discovery: []string{
				provider.DiscoveryGroup,
				provider.DiscoveryProject,
				provider.DiscoveryTerraform,
				provider.DiscoveryK8sManifests,
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
					Desc:    "Custom GitLab base URL (https://example.com/)",
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
