// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ansible/provider"
)

var Config = plugin.Provider{
	Name:            "ansible",
	ID:              "go.mondoo.com/cnquery/v11/providers/ansible",
	Version:         "11.0.35",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ansible",
			Use:   "ansible PATH",
			Short: "an Ansible playbook",
			Long: `Use the ansible provider to query resources in an Ansible playbook.

Example:
  cnquery shell ansible <path>
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=iac", "category=ansible"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"playbook": nil,
			},
		},
	},
}
