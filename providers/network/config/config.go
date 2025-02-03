// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/network/connection"
	"go.mondoo.com/cnquery/v11/providers/network/provider"
)

var Config = plugin.Provider{
	Name:            "network",
	ID:              "go.mondoo.com/cnquery/v9/providers/network",
	Version:         "11.0.51",
	ConnectionTypes: []string{provider.HostConnectionType},
	CrossProviderTypes: []string{
		"go.mondoo.com/cnquery/providers/os",
		"go.mondoo.com/cnquery/providers/k8s",
		"go.mondoo.com/cnquery/providers/aws",
		// FIXME: DEPRECATED, remove in v12.0 vv
		// Until v10 providers had a version indication in their ID. With v10
		// this is no longer the case. Once we get far enough away from legacy
		// version support, we can safely remove this.
		"go.mondoo.com/cnquery/v9/providers/os",
		"go.mondoo.com/cnquery/v9/providers/k8s",
		"go.mondoo.com/cnquery/v9/providers/aws",
		// ^^
	},
	Connectors: []plugin.Connector{
		{
			Name:  "host",
			Use:   "host HOST",
			Short: "a remote HTTP or HTTPS host",
			Long: `Use the host provider to query remote HTTP or HTTPS hosts. 

Examples:
  cnquery shell host <YOUR-DOMAIN-OR-IP>
  cnspec scan host <YOUR-DOMAIN-OR-IP>

Note:
  If you don't provide a protocol, Mondoo assumes HTTPS.
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "insecure",
					Type:    plugin.FlagType_Bool,
					Default: "",
					Desc:    "Disable TLS/SSL verification",
				},
				{
					Long:    connection.OPTION_FOLLOW_REDIRECTS,
					Type:    plugin.FlagType_Bool,
					Default: "",
					Desc:    "Follow HTTP redirects",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=network", "category=host"},
		},
	},
}
