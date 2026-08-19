// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/minio/connection"
	"go.mondoo.com/mql/v13/providers/minio/provider"
)

var Config = plugin.Provider{
	Name:            "minio",
	ID:              "go.mondoo.com/mql/providers/minio",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "minio",
			Use:   "minio",
			Short: "a MinIO object storage deployment",
			Long: `Use the minio provider to query the security posture of a MinIO deployment,
including its buckets, bucket access policies, encryption and retention
settings, users, groups, service accounts, named policies, and audit and
server-log webhooks.

Authenticate with the deployment's root credentials:

  export MINIO_ENDPOINT=https://minio.example.com:9000
  export MINIO_ROOT_USER=minioadmin
  export MINIO_ROOT_PASSWORD=<SECRET>
  mql shell minio

Or pass the key pair on the command line:

  mql shell minio --endpoint https://minio.example.com:9000 \
    --access-key <ACCESS-KEY> --secret-key <SECRET-KEY>

MINIO_ACCESS_KEY and MINIO_SECRET_KEY are read when the MINIO_ROOT_* pair is
not set. A deployment published under a private certificate authority needs
--ca-cert; --tls-skip-verify is for lab deployments only.

The key pair needs admin:ServerInfo, admin:ConfigUpdate (read), admin:ListUsers,
admin:ListGroups, admin:ListServiceAccounts and admin:GetPolicy on the admin
API, plus s3:ListAllMyBuckets and the s3:GetBucket* actions on every bucket.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionEndpoint,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "MinIO S3 and admin API endpoint (defaults to $MINIO_ENDPOINT)",
				},
				{
					Long:    connection.OptionAccessKey,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Access key (defaults to $MINIO_ROOT_USER, then $MINIO_ACCESS_KEY)",
				},
				{
					Long:    "secret-key",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Secret key (defaults to $MINIO_ROOT_PASSWORD, then $MINIO_SECRET_KEY)",
				},
				{
					Long:    connection.OptionRegion,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Region to sign requests for (defaults to $MINIO_REGION)",
				},
				{
					Long:    connection.OptionCACert,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Certificate authority to trust, as a PEM file path (defaults to $MINIO_CACERT)",
				},
				{
					Long:    connection.OptionTLSSkipVerify,
					Type:    plugin.FlagType_Bool,
					Default: "false",
					Desc:    "Skip TLS certificate verification, for lab deployments only",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=minio"},
			Key:          "host",
			Title:        "Host",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
	Platforms: connection.Platforms,
}
