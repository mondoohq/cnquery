// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/consul/connection"
	"go.mondoo.com/mql/v13/providers/consul/provider"
)

var Config = plugin.Provider{
	Name:            "consul",
	ID:              "go.mondoo.com/mql/providers/consul",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "consul",
			Use:   "consul",
			Short: "a HashiCorp Consul agent",
			Long: `Use the consul provider to query the runtime security posture of a
self-managed HashiCorp Consul agent, including its ACL system, gossip
encryption, agent-to-agent TLS verification, and service mesh intentions.

Query the local agent:

  mql shell consul

Or a remote agent, authenticating with an ACL token:

  export CONSUL_HTTP_ADDR=https://consul.example.com:8501
  export CONSUL_HTTP_TOKEN=00000000-0000-0000-0000-000000000000
  mql shell consul

The token needs agent:read to read the agent configuration, acl:read to read
the token, policy and role inventory, and operator:read for the mesh
configuration. Without acl:read the ACL inventory reports an error rather than
an empty list, so a missing permission cannot read as a clean result. See the
provider README for a least-privilege policy.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionAddress,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Consul HTTP API address (defaults to $CONSUL_HTTP_ADDR, then http://127.0.0.1:8500)",
				},
				{
					Long:    "token",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Consul ACL token (defaults to $CONSUL_HTTP_TOKEN)",
				},
				{
					Long:    connection.OptionCACert,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Certificate authority to trust, as a PEM file path (defaults to $CONSUL_CACERT)",
				},
				{
					Long:    connection.OptionTLSSkipVerify,
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Skip TLS certificate verification, for lab agents only",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=consul"},
			Key:          "host",
			Title:        "Host",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": {
					Key:   "datacenter",
					Title: "Datacenter",
					Values: map[string]*inventory.AssetUrlBranch{
						"*": nil,
					},
				},
			},
		},
	},
	Platforms: connection.Platforms,
}
