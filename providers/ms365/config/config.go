// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ms365/provider"
)

var Config = plugin.Provider{
	Name:            "ms365",
	ID:              "go.mondoo.com/cnquery/v9/providers/ms365",
	Version:         "11.1.14",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ms365",
			Use:   "ms365",
			Short: "a Microsoft 365 account",
			Long: `Use the ms365 provider to query resources within Microsoft 365, including organizations, users, roles, SharePoint sites, and more.

Examples:
  cnquery shell  ms365 --certificate-path <PATH-TO-YOUR-PEM> --tenant-id <YOUR-TENANT-ID> --client-id <YOUR-CLIENT-ID>
  cnspec scan ms365 --certificate-path <PATH-TO-YOUR-PEM> --tenant-id <YOUR-TENANT-ID> --client-id <YOUR-CLIENT-ID>

Notes:
  If you give cnquery access through the Microsoft 365 API, you can omit the certificate-path, tenant-id, and client-id flags. To learn how, read https://mondoo.com/docs/cnquery/saas/ms365/#give-cnquery-access-through-the-microsoft-365-api.
`,
			MinArgs:   0,
			MaxArgs:   5,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    "tenant-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Directory (tenant) ID of the service principal",
				},
				{
					Long:    "client-id",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Application (client) ID of the service principal",
				},
				{
					Long:    "organization",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Organization to scan",
				},
				{
					Long:    "sharepoint-url",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Sharepoint URL to scan",
				},
				{
					Long:    "client-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Secret for the application",
				},
				{
					Long:    "certificate-path",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Path (in PKCS #12/PFX or PEM format) to the authentication certificate",
				},
				{
					Long:    "certificate-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Passphrase for the authentication certificate file",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=ms365"},
		},
	},
}
