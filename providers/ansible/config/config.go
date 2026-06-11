// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ansible/provider"
)

var Config = plugin.Provider{
	Name:            "ansible",
	ID:              "go.mondoo.com/mql/v13/providers/ansible",
	Version:         "13.2.2",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ansible",
			Use:   "ansible PATH",
			Short: "an Ansible playbook or project",
			Long: `Use the ansible provider to statically analyze Ansible infrastructure as code.

Point PATH at a single playbook file to query its plays, tasks, handlers, and
variables. Point PATH at a project directory to additionally query roles,
inventory and host/group variables, Galaxy requirements, ansible.cfg, and
vault-encrypted files.

Examples:
  cnspec shell ansible <playbook.yml>
  cnspec shell ansible <project-dir>
  cnspec scan ansible <project-dir>
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
				"project":  nil,
			},
		},
	},
}
