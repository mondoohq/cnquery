// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
	"go.mondoo.com/cnquery/v11/providers/github/provider"
)

var Config = plugin.Provider{
	Name:            "github",
	ID:              "go.mondoo.com/cnquery/v9/providers/github",
	Version:         "11.4.27",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "github",
			Use:   "github",
			Short: "a GitHub organization or repository",
			Long: `Use the github provider to query resources within GitHub organizations and repositories.

Available commands:
  org					              GitHub organization
  repo                      GitHub repo

Examples:
  cnspec scan github org <ORG_NAME> --discover organization
  cnspec scan github org <ORG_NAME> --repos "<REPO1>,<REPO2>"
  cnquery shell github org <ORG_NAME>
	cnquery shell github org <YOUR-GITHUB-ORG> --app-id <YOUR-GITHUB-APP-ID> --app-installation-id <YOUR-GITHUB-APP-INSTALL-ID> --app-private-key <PATH-TO-PEM-FILE>

Notes:
  Mondoo needs a personal access token to scan a GitHub organization, public repo, or private repo. The token's level of access determines how much information Mondoo can retrieve. Supply your personal access token to Mondoo by setting the GITHUB_TOKEN environment variable. To learn how, read https://mondoo.com/docs/cnspec/saas/github/.

	If you have very large GitHub organizations, consider giving Mondoo access using custom GitHub app credentials. To learn how, read https://mondoo.com/docs/cnspec/saas/gh-app/.

	If you have a GitHub Enterprise Server account, you must provide the URL for the account using the --enterprise-url flag.
`,
			MinArgs: 2,
			MaxArgs: 2,
			Discovery: []string{
				connection.DiscoveryRepos,
				connection.DiscoveryUsers,
				connection.DiscoveryOrganization,
				connection.DiscoveryTerraform,
				connection.DiscoveryK8sManifests,
			},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitHub personal access token",
				},
				{
					Long:    "repos-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Filter out repositories matching these names",
				},
				{
					Long:    "repos",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Only include repositories matching these names",
				},
				{
					Long:    "app-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitHub App ID",
				},
				{
					Long:    "app-installation-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitHub App installation ID",
				},
				{
					Long:    "app-private-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitHub App private key file path",
				},
				{
					Long:    "enterprise-url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "GitHub Enterprise Server URL",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=github"},
			Key:          "organization",
			Title:        "Organization",
			Values: map[string]*inventory.AssetUrlBranch{
				"organization": {
					Key:   "organization",
					Title: "Organization",
					Values: map[string]*inventory.AssetUrlBranch{
						"organization": nil,
						"*": {
							Key:   "repository",
							Title: "Repository",
							Values: map[string]*inventory.AssetUrlBranch{
								"*": nil,
							},
						},
					},
				},
				"user": {
					Key:   "user",
					Title: "User",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": nil,
					},
				},
			},
		},
	},
}
