// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
	"go.mondoo.com/mql/v13/providers/confluent/provider"
)

var Config = plugin.Provider{
	Name:            "confluent",
	ID:              "go.mondoo.com/mql/providers/confluent",
	Version:         "13.0.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "confluent",
			Use:   "confluent",
			Short: "a Confluent Cloud organization",
			Long: `Use the confluent provider to query the security posture of a Confluent Cloud
organization, including its environments, Kafka clusters and their network
exposure, topic ACLs, RBAC role bindings, service accounts, API keys, Schema
Registry clusters, self-managed encryption keys, and audit log destination.

Authenticate with a Cloud API key:

  export CONFLUENT_CLOUD_API_KEY=<KEY>
  export CONFLUENT_CLOUD_API_SECRET=<SECRET>
  mql shell confluent

Topics and ACLs are served by each cluster's own REST endpoint, which only
accepts a cluster-scoped Kafka API key. Supply one to read them:

  export CONFLUENT_KAFKA_API_KEY=<KEY>
  export CONFLUENT_KAFKA_API_SECRET=<SECRET>

With more than one cluster, set a pair per cluster by appending the cluster ID
in upper case with hyphens replaced by underscores, for example
CONFLUENT_KAFKA_API_KEY_LKC_ABC123.

The Cloud API key needs the OrganizationAdmin role, or a combination of reader
roles covering every environment you want to scan.
`,
			Discovery: []string{},
			Flags: []plugin.Flag{
				{
					Long:    connection.OptionAPIKey,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Cloud API key (defaults to $CONFLUENT_CLOUD_API_KEY)",
				},
				{
					Long:    "api-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Cloud API secret (defaults to $CONFLUENT_CLOUD_API_SECRET)",
				},
				{
					Long:    connection.OptionKafkaAPIKey,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Cluster-scoped Kafka API key, needed for topics and ACLs (defaults to $CONFLUENT_KAFKA_API_KEY)",
				},
				{
					Long:    "kafka-api-secret",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Cluster-scoped Kafka API secret (defaults to $CONFLUENT_KAFKA_API_SECRET)",
				},
				{
					Long:    connection.OptionBaseURL,
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Management API root, for deployments not served from api.confluent.cloud",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=saas", "provider=confluent"},
			Key:          "organization",
			Title:        "Organization",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
	Platforms: connection.Platforms,
}
