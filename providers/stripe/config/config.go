// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
	"go.mondoo.com/mql/v13/providers/stripe/provider"
)

var Config = plugin.Provider{
	Name:            "stripe",
	ID:              "go.mondoo.com/mql/providers/stripe",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "stripe",
			Use:   "stripe",
			Short: "a Stripe account",
			Long: `Use the stripe provider to query the configuration and security posture of your Stripe account.

Examples:
  cnspec shell stripe --token <secret_key>
  cnspec scan stripe --token <secret_key>

Notes:
  If you set the STRIPE_API_KEY environment variable, you can omit the token flag.
  A read-only restricted key (rk_...) is sufficient and recommended.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Stripe secret key for authentication (or set STRIPE_API_KEY)",
				},
				{
					Long:    "account",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Optional connected account ID to scope the connection to (Stripe Connect)",
				},
			},
		},
	},
}
