// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/cloudformation/provider"
)

var Config = plugin.Provider{
	Name:            "cloudformation",
	ID:              "go.mondoo.com/cnquery/v11/providers/cloudformation",
	Version:         "11.0.18",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:      "cloudformation",
			Use:       "cloudformation PATH",
			Short:     "an AWS CloudFormation template or AWS SAM template",
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=iac", "category=cloudformation"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"template": nil,
			},
		},
	},
}
