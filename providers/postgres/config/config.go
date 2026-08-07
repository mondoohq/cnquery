// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/postgres/connection"
	"go.mondoo.com/mql/v13/providers/postgres/provider"
)

var Config = plugin.Provider{
	Name:            "postgres",
	ID:              "go.mondoo.com/mql/v13/providers/postgres",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Platforms:       connection.Platforms,
	Connectors: []plugin.Connector{
		{
			Name:  "postgres",
			Use:   "postgres [host]",
			Short: "a PostgreSQL server",
			Long: `Use the postgres provider to query a PostgreSQL server.

The provider connects to a PostgreSQL server and runs read-only catalog
queries (pg_catalog and information_schema) to inventory roles, databases,
schemas, privileges, settings, host-based authentication rules, extensions,
foreign servers, and replication configuration.

By default the provider discovers every connectable database on the server as
its own asset.

Examples:
  cnspec shell postgres db.contoso.com --user postgres --ask-pass
  cnspec scan postgres db.contoso.com --database appdb --sslmode verify-full --sslrootcert ca.pem
`,
			Discovery: []string{
				connection.DiscoveryAuto,
				connection.DiscoveryAll,
				connection.DiscoveryInstance,
				connection.DiscoveryDatabases,
				connection.DiscoveryNone,
			},
			Flags: []plugin.Flag{
				{
					Long:    "host",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "PostgreSQL hostname or IP address",
				},
				{
					Long:    "port",
					Type:    plugin.FlagType_Int,
					Default: "5432",
					Desc:    "PostgreSQL TCP port",
				},
				{
					Long:    "database",
					Type:    plugin.FlagType_String,
					Default: "postgres",
					Desc:    "Database to connect to for the initial (server) connection",
				},
				{
					Long:    "user",
					Short:   "u",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Role name to authenticate as",
				},
				{
					Long:        "password",
					Short:       "p",
					Type:        plugin.FlagType_String,
					Default:     "",
					Desc:        "Set the connection password",
					Option:      plugin.FlagOption_Password,
					ConfigEntry: "-",
				},
				{
					Long:        "ask-pass",
					Type:        plugin.FlagType_Bool,
					Default:     "false",
					Desc:        "Prompt for the connection password",
					ConfigEntry: "-",
				},
				{
					Long:    "sslmode",
					Type:    plugin.FlagType_String,
					Default: "prefer",
					Desc:    "TLS mode: disable, allow, prefer, require, verify-ca, or verify-full",
				},
				{
					Long:    "sslrootcert",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to the trusted CA certificate for TLS verification",
				},
				{
					Long:    "sslcert",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to the client certificate for TLS client authentication",
				},
				{
					Long:    "sslkey",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path to the client private key for TLS client authentication",
				},
			},
		},
	},
}
