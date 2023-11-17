// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/terraform/provider"
)

var Config = plugin.Provider{
	Name:    "terraform",
	ID:      "go.mondoo.com/cnquery/v9/providers/terraform",
	Version: "9.1.3",
	ConnectionTypes: []string{
		provider.StateConnectionType,
		provider.PlanConnectionType,
		provider.HclConnectionType,
		provider.HclGitConnectionType,
	},
	Connectors: []plugin.Connector{
		{
			Name:      "terraform",
			Aliases:   []string{},
			Use:       "terraform PATH",
			Short:     "a Terraform HCL file or directory",
			MinArgs:   1,
			MaxArgs:   2,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
