// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/provider"
)

var Config = plugin.Provider{
	Name:    "terraform",
	ID:      "go.mondoo.com/cnquery/v9/providers/terraform",
	Version: "11.1.86",
	ConnectionTypes: []string{
		provider.StateConnectionType,
		provider.PlanConnectionType,
		provider.HclConnectionType,
		provider.HclGitConnectionType,
	},
	Connectors: []plugin.Connector{
		{
			Name:    "terraform",
			Aliases: []string{},
			Use:     "terraform PATH",
			Short:   "Terraform HCL configurations, plan files, and state files",
			Long: `Use the terraform provider to query Terraform HCL, plan, or state files as well as directories of files.

Available commands:
  plan                       Terraform plan file
  state                      Terraform state file

Examples:
  cnquery shell terraform <PATH-TO-HCL-DIRECTORY>
  cnspec scan terraform <PATH-TO-HCL-FILE>
  cnspec scan terraform plan <PATH-TO-PLAN-JSON>
  cnspec scan terraform state <PATH-TO-STATE-JSON>
`,
			MinArgs:   1,
			MaxArgs:   2,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:        "ignore-dot-terraform",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Exclude the .terraform directory (contains cached provider plugins and modules)",
					ConfigEntry: "ignore_dot_terraform",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=iac", "category=terraform"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"hcl":   nil,
				"plan":  nil,
				"state": nil,
			},
		},
	},
}
