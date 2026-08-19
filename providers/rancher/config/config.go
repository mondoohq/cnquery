// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/rancher/connection"
	"go.mondoo.com/mql/v13/providers/rancher/provider"
)

var Config = plugin.Provider{
	Name:            "rancher",
	ID:              "go.mondoo.com/mql/providers/rancher",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "rancher",
			Use:   "rancher",
			Short: "a Rancher Manager fleet",
			Long: `Use the rancher provider to query the multi-cluster governance layer of a
Rancher Manager install: the clusters it manages, the global and project role
templates and bindings that decide who can administer them, the cluster and pod
security admission templates that constrain how clusters may be built, the
authentication providers that let people in, and the API tokens and registry
credentials in circulation.

Per-cluster workloads are not queried here. Point the k8s provider at a
downstream cluster for those.

Authenticate with an API token:

  export RANCHER_URL=https://rancher.example.com
  export RANCHER_TOKEN=token-abcde:xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  mql shell rancher

Or with the two halves of the key separately:

  mql shell rancher --url https://rancher.example.com \
    --access-key token-abcde --secret-key <SECRET>

Rancher is commonly published under a private certificate authority. Trust it
with --ca-cert rather than turning verification off:

  mql shell rancher --ca-cert /etc/ssl/certs/rancher-ca.pem

The token needs read access across the management API. A restricted-admin or a
custom global role granting get and list on the management.cattle.io resources
is enough; a standard user sees only the clusters and projects they are a
member of, and only their own tokens.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionURL,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Rancher Manager URL (defaults to $RANCHER_URL)",
				},
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Rancher API token, as access-key:secret-key (defaults to $RANCHER_TOKEN)",
				},
				{
					Long:    connection.OptionAccessKey,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Rancher API access key, used with --secret-key instead of a token (defaults to $RANCHER_ACCESS_KEY)",
				},
				{
					Long:    "secret-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Rancher API secret key (defaults to $RANCHER_SECRET_KEY)",
				},
				{
					Long:    connection.OptionCACert,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Certificate authority to trust, as a PEM file path (defaults to $RANCHER_CACERT)",
				},
				{
					Long:    connection.OptionTLSSkipVerify,
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Skip TLS certificate verification, for lab servers only",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=rancher"},
			Key:          "host",
			Title:        "Host",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
	Platforms: connection.Platforms,
}
