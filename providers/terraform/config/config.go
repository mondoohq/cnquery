// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "terraform",
	ID:      "go.mondoo.com/cnquery/providers/terraform",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "terraform",
			Aliases:   []string{},
			Use:       "terraform PATH",
			Short:     "a terraform hcl file or directory.",
			MinArgs:   1,
			MaxArgs:   2,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
