// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package config

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/tailscale/connection"
	"go.mondoo.com/cnquery/v11/providers/tailscale/provider"
)

var Config = plugin.Provider{
	Name:            "tailscale",
	ID:              "go.mondoo.com/cnquery/v11/providers/tailscale",
	Version:         "11.0.3",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "tailscale",
			Use:   "tailscale",
			Short: "a Tailscale network",
			// The tailnet organization name. e.g. example.com
			MinArgs: 0,
			MaxArgs: 1,
			Long: fmt.Sprintf(`
Use the tailscale provider to query devices, DNS nameservers, and more information about a Tailscale network,
known as a tailnet.

To authenticate using an API access token:

  cnquery shell tailscale --token <access-token>

To authenticate using an OAuth client:

  cnquery shell tailscale --client-id <id> --client-secret <secret>

You can also use the default environment variables '%s', '%s',
and '%s' to provide your credentials.

If you are using an API access token instead of an OAuth client, use the '%s' variable instead.
`,
				connection.TAILSCALE_OAUTH_CLIENT_ID_VAR,
				connection.TAILSCALE_OAUTH_CLIENT_SECRET_VAR,
				connection.TAILSCALE_TAILNET_VAR,
				connection.TAILSCALE_API_KEY_VAR,
			),
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryAuto,
				connection.DiscoveryDevices,
				connection.DiscoveryUsers,
			},
			Flags: []plugin.Flag{
				{
					Long:    connection.OPTION_TOKEN,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Tailscale access token",
				},
				{
					Long:    connection.OPTION_BASE_URL,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Base URL for the Tailscale API",
				},
				{
					Long:    connection.OPTION_CLIENT_ID,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "OAuth client ID",
				},
				{
					Long:    connection.OPTION_CLIENT_SECRET,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "OAuth client secret",
				},
			},
		},
	},
}
