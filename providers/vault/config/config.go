// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vault/connection"
	"go.mondoo.com/mql/v13/providers/vault/provider"
)

var Config = plugin.Provider{
	Name:            "vault",
	ID:              "go.mondoo.com/mql/providers/vault",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "vault",
			Use:   "vault",
			Short: "a HashiCorp Vault server",
			Long: `Use the vault provider to query the security posture of a HashiCorp Vault
server, including its seal state, audit devices, authentication methods, ACL
policies, secret engines, and Enterprise namespaces.

Authenticate with a token:

  export VAULT_ADDR=https://vault.example.com:8200
  export VAULT_TOKEN=hvs.example
  mql shell vault

Or with AppRole:

  mql shell vault --address https://vault.example.com:8200 \
    --role-id <ROLE-ID> --secret-id <SECRET-ID>

The token needs read access to the sys/ endpoints this provider queries:
sys/health, sys/seal-status, sys/audit (with sudo), sys/auth, sys/mounts,
sys/policies/acl, and sys/namespaces on Enterprise. See the provider README for
a least-privilege policy.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionAddress,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Vault API address (defaults to $VAULT_ADDR)",
				},
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Vault token (defaults to $VAULT_TOKEN)",
				},
				{
					Long:    connection.OptionRoleID,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "AppRole role ID, used with --secret-id instead of a token",
				},
				{
					Long:    "secret-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "AppRole secret ID (defaults to $VAULT_SECRET_ID)",
				},
				{
					Long:    connection.OptionNamespace,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Enterprise namespace to scan (defaults to $VAULT_NAMESPACE)",
				},
				{
					Long:    connection.OptionCACert,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Certificate authority to trust, as a PEM file path (defaults to $VAULT_CACERT)",
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
			PathSegments: []string{"technology=saas", "provider=vault"},
			Key:          "host",
			Title:        "Host",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": {
					Key:   "namespace",
					Title: "Namespace",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": nil,
					},
				},
			},
		},
	},
	Platforms: connection.Platforms,
}
