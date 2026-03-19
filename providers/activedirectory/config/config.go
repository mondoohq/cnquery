// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/activedirectory/provider"
)

var Config = plugin.Provider{
	Name:            "activedirectory",
	ID:              "go.mondoo.com/cnquery/v9/providers/activedirectory",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{{
		Name:    "activedirectory",
		Use:     "activedirectory",
		Aliases: []string{"ad"},
		Short:   "an Active Directory domain",
		Long: `Use the activedirectory provider to query Active Directory Domain Services via LDAP.

Examples:
  cnspec shell activedirectory --dc dc01.corp.local --user admin@corp.local --password <PASSWORD>
  cnspec scan activedirectory --dc dc01.corp.local --user admin@corp.local --password <PASSWORD>

Notes:
  The provider connects via LDAP (port 389) or LDAPS (port 636). All authenticated domain users can query most AD data.
`,
		MinArgs:   0,
		MaxArgs:   0,
		Discovery: []string{},
		Flags: []plugin.Flag{
			{Long: "dc", Type: plugin.FlagType_String, Desc: "Domain controller hostname or IP address"},
			{Long: "user", Type: plugin.FlagType_String, Desc: "Username for LDAP bind (user@domain.com or DOMAIN\\user)"},
			{Long: "password", Type: plugin.FlagType_String, Desc: "Password for LDAP bind"},
			{Long: "domain", Type: plugin.FlagType_String, Desc: "Domain DNS name (auto-detected from RootDSE if omitted)"},
			{Long: "base-dn", Type: plugin.FlagType_String, Desc: "Base DN for LDAP searches (auto-detected from RootDSE if omitted)"},
			{Long: "ldaps", Type: plugin.FlagType_Bool, Desc: "Use LDAPS (TLS, port 636) instead of plain LDAP"},
			{Long: "port", Type: plugin.FlagType_Int, Desc: "LDAP port (default: 389 for LDAP, 636 for LDAPS)"},
			{Long: "insecure", Type: plugin.FlagType_Bool, Desc: "Skip TLS certificate verification"},
			{Long: "backend", Type: plugin.FlagType_String, Default: "ldap", Desc: "Backend to use: ldap (default) or rsat (Windows only, requires RSAT AD DS tools)"},
		},
	}},
	AssetUrlTrees: []*inventory.AssetUrlBranch{{
		PathSegments: []string{"technology=directory-service", "provider=activedirectory"},
	}},
}
