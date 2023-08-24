// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

var Config = plugin.Provider{
	Name:    "gcp",
	ID:      "go.mondoo.com/cnquery/providers/gcp",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:      "gcp",
			Use:       "gcp",
			Short:     "GCP Cloud",
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
