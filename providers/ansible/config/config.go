// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ansible/provider"
)

var Config = plugin.Provider{
	Name:            "ansible",
	ID:              "go.mondoo.com/cnquery/v11/providers/ansible",
	Version:         "11.0.8",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "ansible",
			Use:       "ansible PATH",
			Short:     "an Ansible playbook",
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
