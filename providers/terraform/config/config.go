// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/terraform/provider"
)

var Config = plugin.Provider{
	Name: "terraform",
	// Every kind this provider hands out as its own asset is a root (ADR 031).
	Root:    "terraform",
	ID:      "go.mondoo.com/mql/providers/terraform",
	Version: "13.3.13",
	// Every root carries `asset`, which core owns (ADR 042).
	Requires: []plugin.ProviderDep{
		{ID: "go.mondoo.com/mql/providers/core", Name: "core", MinVersion: "13.0.0"},
	},
	Platforms: provider.Platforms,
	ConnectionTypes: []string{
		provider.StateConnectionType,
		provider.PlanConnectionType,
		provider.HclConnectionType,
		provider.HclGitConnectionType,
	},
	Connectors: []plugin.Connector{
		{
			Name:    "terraform",
			Aliases: []string{"opentofu", "tofu"},
			Use:     "terraform PATH",
			Short:   "Terraform and OpenTofu HCL configurations, plan files, and state files",
			Long: `Use the terraform provider to query Terraform or OpenTofu HCL, plan, or state files as well as directories of files.

OpenTofu configurations are read as well as Terraform ones. In a directory
holding both, the OpenTofu file wins for any name it shares with a Terraform
file, matching how OpenTofu itself loads a configuration: main.tofu replaces
main.tf, and .tofu.json, .tofuvars and .tofuvars.json replace their .tf
equivalents in the same way.

For HCL the tool is detected from the files present. Plan and state files carry
no marker of their own -- their JSON is identical between the two tools -- so
they are reported as Terraform unless --iac-tool says otherwise.

Available commands:
  plan                       Terraform or OpenTofu plan file
  state                      Terraform or OpenTofu state file

Examples:
  cnspec shell terraform <PATH-TO-HCL-DIRECTORY>
  cnspec scan terraform <PATH-TO-HCL-FILE>
  cnspec scan terraform plan <PATH-TO-PLAN-JSON>
  cnspec scan terraform state <PATH-TO-STATE-JSON> --iac-tool opentofu
  cnspec scan terraform <PATH-TO-MIXED-DIRECTORY> --iac-tool terraform
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
				{
					Long:    "iac-tool",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Read the configuration as \"terraform\" or \"opentofu\" instead of detecting it from the files present",
					// "-" reads the flag straight from the command line. A named
					// config entry would be bound into viper under that name but
					// read back under the flag's own name, so the value the user
					// typed never arrives.
					ConfigEntry: "-",
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
