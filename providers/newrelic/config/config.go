// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/newrelic/connection"
	"go.mondoo.com/mql/v13/providers/newrelic/provider"
)

var Config = plugin.Provider{
	Name:            "newrelic",
	ID:              "go.mondoo.com/mql/providers/newrelic",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "newrelic",
			Use:   "newrelic",
			Short: "a New Relic account",
			Long: `Use the newrelic provider to query the security posture of a New Relic
account and the organization it belongs to, including users, groups, roles and
account access grants, authentication domains and their single sign-on and SCIM
settings, user and ingest keys, alert policies and notification destinations,
and the drop rules and retention rules that decide which telemetry survives.

Authenticate with a New Relic user key:

  export NEW_RELIC_API_KEY=NRAK-EXAMPLE
  export NEW_RELIC_ACCOUNT_ID=1234567
  mql shell newrelic

Or pass the same values as flags, selecting the EU data region:

  mql shell newrelic --api-key NRAK-EXAMPLE --account-id 1234567 --region eu

The key must be a user key (the NRAK- prefix), not a license key. Reading the
organization-wide resources (users, groups, roles, access grants and
authentication domains) needs a key belonging to a user with organization
read permissions. Key material is never read: the provider does not request the
keystring of any key it reports.
`,
			MinArgs:   0,
			MaxArgs:   0,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "api-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "New Relic user API key (defaults to $NEW_RELIC_API_KEY)",
				},
				{
					Long:    connection.OptionAccountID,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "New Relic account ID to scan (defaults to $NEW_RELIC_ACCOUNT_ID)",
				},
				{
					Long:    connection.OptionRegion,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "New Relic data region, us or eu (defaults to $NEW_RELIC_REGION, then us)",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=newrelic"},
			Key:          "region",
			Title:        "Region",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": {
					Key:   "account",
					Title: "Account",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": nil,
					},
				},
			},
		},
	},
	Platforms: connection.Platforms,
}
