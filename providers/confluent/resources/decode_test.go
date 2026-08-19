// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The payloads below are shaped like the documented responses of the Confluent
// Cloud management API and the per-cluster Kafka REST API. A mistyped struct tag
// yields a zero value rather than an error, so every security-relevant field is
// pinned against a payload rather than trusted to the compiler.

func TestDecodeOrganization(t *testing.T) {
	const payload = `{
      "api_version": "org/v2",
      "kind": "Organization",
      "id": "9bb441c4-edef-46ac-8a41-c49e44a3fd9a",
      "metadata": {
        "self": "https://api.confluent.cloud/org/v2/organizations/9bb441c4-edef-46ac-8a41-c49e44a3fd9a",
        "resource_name": "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a",
        "created_at": "2021-03-01T10:00:00Z"
      },
      "display_name": "Acme Streaming",
      "jit_enabled": true
    }`

	var record OrganizationRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "9bb441c4-edef-46ac-8a41-c49e44a3fd9a", record.ID)
	assert.Equal(t, "Acme Streaming", record.DisplayName)
	require.NotNil(t, record.JitEnabled)
	assert.True(t, *record.JitEnabled)
	assert.Equal(t, "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a", record.Metadata.ResourceName)
	require.NotNil(t, record.Metadata.CreatedAt.Time())
}

// An organization that does not report the flag must leave it absent rather
// than decoding to false, which would say the feature is switched off.
func TestDecodeOrganizationWithoutJitFlag(t *testing.T) {
	var record OrganizationRecord
	require.NoError(t, json.Unmarshal([]byte(`{"id":"o-1","display_name":"x"}`), &record))
	assert.Nil(t, record.JitEnabled)
}

func TestDecodeEnvironment(t *testing.T) {
	const payload = `{
      "api_version": "org/v2",
      "kind": "Environment",
      "id": "env-abc123",
      "metadata": {
        "self": "https://api.confluent.cloud/org/v2/environments/env-abc123",
        "resource_name": "crn://confluent.cloud/organization=o-1/environment=env-abc123",
        "created_at": "2022-05-04T09:11:00Z",
        "updated_at": "2023-01-02T03:04:05Z"
      },
      "display_name": "production",
      "stream_governance_config": {"package": "ADVANCED"}
    }`

	var record environmentRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "env-abc123", record.ID)
	assert.Equal(t, "production", record.DisplayName)
	require.NotNil(t, record.StreamGovernanceConfig)
	assert.Equal(t, "ADVANCED", record.StreamGovernanceConfig.Package)
	assert.Equal(t, "crn://confluent.cloud/organization=o-1/environment=env-abc123", record.Metadata.ResourceName)
	require.NotNil(t, record.Metadata.UpdatedAt.Time())
	assert.Equal(t, 2023, record.Metadata.UpdatedAt.Time().Year())

	var bare environmentRecord
	require.NoError(t, json.Unmarshal([]byte(`{"id":"env-1"}`), &bare))
	assert.Nil(t, bare.StreamGovernanceConfig, "an environment without a package must not report one")
	assert.Nil(t, bare.Metadata.CreatedAt.Time(), "an absent timestamp must stay null, not become the zero time")
}

func TestDecodeKafkaClusterDedicatedWithByok(t *testing.T) {
	const payload = `{
      "api_version": "cmk/v2",
      "kind": "Cluster",
      "id": "lkc-abc123",
      "metadata": {
        "self": "https://api.confluent.cloud/cmk/v2/clusters/lkc-abc123",
        "resource_name": "crn://confluent.cloud/organization=o-1/environment=env-1/cloud-cluster=lkc-abc123",
        "created_at": "2024-02-03T04:05:06Z"
      },
      "spec": {
        "display_name": "prod-kafka",
        "availability": "MULTI_ZONE",
        "cloud": "AWS",
        "region": "us-east-1",
        "config": {"kind": "Dedicated", "cku": 2, "zones": ["use1-az1", "use1-az2", "use1-az4"]},
        "endpoints": {
          "PUBLIC": {
            "kafka_bootstrap_endpoint": "lkc-abc123-0000.us-east-1.aws.glb.confluent.cloud:9092",
            "http_endpoint": "https://lkc-abc123-0000.us-east-1.aws.glb.confluent.cloud:443",
            "connection_type": "PUBLIC"
          },
          "ap1pni123": {
            "kafka_bootstrap_endpoint": "lkc-abc123-0000.us-east-1.aws.private.confluent.cloud:9092",
            "http_endpoint": "https://lkc-abc123.us-east-1.aws.private.confluent.cloud:443",
            "connection_type": "PRIVATE_NETWORK_INTERFACE"
          }
        },
        "deletion_protection": true,
        "environment": {"id": "env-1", "related": "https://api.confluent.cloud/org/v2/environments/env-1", "resource_name": "crn://..."},
        "network": {"id": "n-xyz", "environment": "env-1", "related": "https://...", "resource_name": "crn://..."},
        "byok": {"id": "cck-key1", "related": "https://...", "resource_name": "crn://..."}
      },
      "status": {"phase": "PROVISIONED", "cku": 2}
    }`

	var record kafkaClusterRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	require.NotNil(t, record.Spec)
	assert.Equal(t, "lkc-abc123", record.ID)
	assert.Equal(t, "prod-kafka", record.Spec.DisplayName)
	assert.Equal(t, "MULTI_ZONE", record.Spec.Availability)
	assert.Equal(t, "AWS", record.Spec.Cloud)
	assert.Equal(t, "us-east-1", record.Spec.Region)
	require.NotNil(t, record.Spec.Config)
	assert.Equal(t, "Dedicated", record.Spec.Config.Kind)
	require.NotNil(t, record.Spec.Config.Cku)
	assert.EqualValues(t, 2, *record.Spec.Config.Cku)
	assert.Nil(t, record.Spec.Config.MaxEcku)
	assert.Equal(t, []string{"use1-az1", "use1-az2", "use1-az4"}, record.Spec.Config.Zones)
	require.NotNil(t, record.Spec.DeletionProtection)
	assert.True(t, *record.Spec.DeletionProtection)
	assert.Equal(t, "env-1", refID(record.Spec.Environment))
	assert.Equal(t, "n-xyz", refID(record.Spec.Network))
	assert.Equal(t, "cck-key1", refID(record.Spec.Byok))
	require.NotNil(t, record.Status)
	assert.Equal(t, "PROVISIONED", record.Status.Phase)

	views := endpointViews(record.Spec.Endpoints)
	require.Len(t, views, 2)
	assert.True(t, hasPublicEndpoint(views))
	assert.True(t, hasPrivateEndpoint(views))
	assert.Equal(t, []string{"PRIVATE_NETWORK_INTERFACE", "PUBLIC"}, endpointConnectionTypes(views))
}

// A cluster reachable only over private link must not decode as public, which
// is the single most consequential reading in this schema.
func TestDecodeKafkaClusterPrivateLinkOnly(t *testing.T) {
	const payload = `{
      "id": "lkc-private",
      "spec": {
        "display_name": "private-kafka",
        "config": {"kind": "Enterprise", "max_ecku": 4},
        "endpoints": {
          "PRIVATE_LINK": {
            "kafka_bootstrap_endpoint": "lkc-private-0000.us-central1.gcp.glb.confluent.cloud:9092",
            "http_endpoint": "https://lkc-private-0000.us-central1.gcp.glb.confluent.cloud",
            "connection_type": "PRIVATE_LINK"
          }
        },
        "deletion_protection": false
      },
      "status": {"phase": "PROVISIONED"}
    }`

	var record kafkaClusterRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	require.NotNil(t, record.Spec.Config)
	assert.Equal(t, "Enterprise", record.Spec.Config.Kind)
	require.NotNil(t, record.Spec.Config.MaxEcku)
	assert.EqualValues(t, 4, *record.Spec.Config.MaxEcku)
	assert.Nil(t, record.Spec.Config.Cku)

	views := endpointViews(record.Spec.Endpoints)
	assert.False(t, hasPublicEndpoint(views))
	assert.True(t, hasPrivateEndpoint(views))
	assert.Empty(t, refID(record.Spec.Byok), "a cluster without a self-managed key reports none")
}

// A response that predates the endpoints map still carries a single endpoint
// pair at the top of the spec, which is the only endpoint information it holds.
func TestDecodeKafkaClusterLegacyEndpoints(t *testing.T) {
	const payload = `{
      "id": "lkc-old",
      "spec": {
        "display_name": "old-kafka",
        "config": {"kind": "Basic"},
        "kafka_bootstrap_endpoint": "SASL_SSL://pkc-1.us-east-1.aws.confluent.cloud:9092",
        "http_endpoint": "https://pkc-1.us-east-1.aws.confluent.cloud:443"
      }
    }`

	var record kafkaClusterRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Empty(t, record.Spec.Endpoints)
	assert.Equal(t, "SASL_SSL://pkc-1.us-east-1.aws.confluent.cloud:9092", record.Spec.LegacyBootstrapEndpoint)
	assert.Equal(t, "https://pkc-1.us-east-1.aws.confluent.cloud:443", record.Spec.LegacyHTTPEndpoint)
}

func TestDecodeTopic(t *testing.T) {
	const payload = `{
      "kind": "KafkaTopic",
      "metadata": {"self": "https://pkc-1.../topics/payments", "resource_name": "crn:///kafka=lkc-1/topic=payments"},
      "cluster_id": "lkc-abc123",
      "topic_name": "payments",
      "is_internal": false,
      "replication_factor": 3,
      "partitions_count": 6
    }`

	var record topicRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "lkc-abc123", record.ClusterID)
	assert.Equal(t, "payments", record.TopicName)
	assert.False(t, record.IsInternal)
	assert.EqualValues(t, 3, record.ReplicationFactor)
	assert.EqualValues(t, 6, record.PartitionsCount)

	var internal topicRecord
	require.NoError(t, json.Unmarshal([]byte(`{"topic_name":"__consumer_offsets","is_internal":true,"replication_factor":3,"partitions_count":50}`), &internal))
	assert.True(t, internal.IsInternal)
}

func TestDecodeTopicConfig(t *testing.T) {
	const payload = `{
      "kind": "KafkaTopicConfig",
      "cluster_id": "lkc-abc123",
      "name": "min.insync.replicas",
      "value": "2",
      "is_default": false,
      "is_read_only": false,
      "is_sensitive": false,
      "source": "DYNAMIC_TOPIC_CONFIG"
    }`

	var record topicConfigRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "min.insync.replicas", record.Name)
	require.NotNil(t, record.Value)
	assert.Equal(t, "2", *record.Value)
	assert.False(t, record.IsSensitive)
	assert.Equal(t, "DYNAMIC_TOPIC_CONFIG", record.Source)

	// A sensitive configuration comes back with a null value, which must stay
	// distinguishable from a configuration set to the empty string.
	var sensitive topicConfigRecord
	require.NoError(t, json.Unmarshal([]byte(`{"name":"ssl.key.password","value":null,"is_sensitive":true}`), &sensitive))
	assert.Nil(t, sensitive.Value)
	assert.True(t, sensitive.IsSensitive)
}

func TestDecodeAcl(t *testing.T) {
	const payload = `{
      "kind": "KafkaAcl",
      "metadata": {"self": "https://pkc-1.../acls"},
      "cluster_id": "lkc-abc123",
      "resource_type": "TOPIC",
      "resource_name": "*",
      "pattern_type": "LITERAL",
      "principal": "User:sa-abc123",
      "host": "*",
      "operation": "READ",
      "permission": "ALLOW"
    }`

	var record aclRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "lkc-abc123", record.ClusterID)
	assert.Equal(t, "TOPIC", record.ResourceType)
	assert.Equal(t, "*", record.ResourceName)
	assert.Equal(t, "LITERAL", record.PatternType)
	assert.Equal(t, "User:sa-abc123", record.Principal)
	assert.Equal(t, "*", record.Host)
	assert.Equal(t, "READ", record.Operation)
	assert.Equal(t, "ALLOW", record.Permission)
}

func TestDecodeApiKey(t *testing.T) {
	const payload = `{
      "api_version": "iam/v2",
      "kind": "ApiKey",
      "id": "ABCDEFGHIJKLMNOP",
      "metadata": {
        "self": "https://api.confluent.cloud/iam/v2/api-keys/ABCDEFGHIJKLMNOP",
        "resource_name": "crn://confluent.cloud/organization=o-1/api-key=ABCDEFGHIJKLMNOP",
        "created_at": "2023-06-01T00:00:00Z"
      },
      "spec": {
        "secret": "SUPER-SECRET-VALUE",
        "display_name": "ci-producer",
        "description": "used by CI",
        "owner": {
          "id": "sa-abc123",
          "related": "https://api.confluent.cloud/iam/v2/service-accounts/sa-abc123",
          "resource_name": "crn://confluent.cloud/service-account=sa-abc123",
          "api_version": "iam/v2",
          "kind": "ServiceAccount"
        },
        "resource": {
          "id": "lkc-abc123",
          "related": "https://api.confluent.cloud/cmk/v2/clusters/lkc-abc123",
          "resource_name": "crn://...",
          "api_version": "cmk/v2",
          "kind": "Cluster"
        }
      }
    }`

	var record apiKeyRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	require.NotNil(t, record.Spec)
	assert.Equal(t, "ABCDEFGHIJKLMNOP", record.ID)
	assert.Equal(t, "ci-producer", record.Spec.DisplayName)
	assert.Equal(t, "used by CI", record.Spec.Description)
	assert.Equal(t, "sa-abc123", refID(record.Spec.Owner))
	assert.Equal(t, "ServiceAccount", ownerKindOf(record.Spec.Owner))
	assert.Equal(t, "lkc-abc123", refID(record.Spec.scopedResource()))
	assert.Equal(t, "cmk.v2.Cluster", referenceKind(record.Spec.scopedResource()))

	// The key secret must never reach any field this provider exposes. Nothing
	// in the decoded record may carry it, so re-encoding the record is a
	// complete sweep of everything the resource layer can read.
	encoded, err := json.Marshal(record)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "SUPER-SECRET-VALUE")
}

// A Cloud API key carries no scoped resource at all, which is what separates an
// organization-wide credential from a cluster-scoped one.
func TestDecodeCloudApiKey(t *testing.T) {
	const payload = `{
      "id": "CLOUDKEY",
      "spec": {
        "display_name": "org admin key",
        "owner": {"id": "u-abc123", "related": "https://...", "resource_name": "crn://...", "api_version": "iam/v2", "kind": "User"},
        "resource": null
      }
    }`

	var record apiKeyRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Nil(t, record.Spec.scopedResource())
	assert.Equal(t, "User", ownerKindOf(record.Spec.Owner))
	assert.Empty(t, referenceKind(record.Spec.scopedResource()))
}

// Newer responses carry a `resources` list alongside the singular reference,
// and a key that only fills the list must not read as a Cloud API key.
func TestDecodeApiKeyWithResourcesList(t *testing.T) {
	const payload = `{
      "id": "KEY2",
      "spec": {
        "owner": {"id": "sa-1", "related": "", "resource_name": ""},
        "resources": [
          {"id": "lsrc-1", "related": "", "resource_name": "", "api_version": "srcm/v3", "kind": "Cluster"}
        ]
      }
    }`

	var record apiKeyRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "lsrc-1", refID(record.Spec.scopedResource()))
	assert.Equal(t, "srcm.v3.Cluster", referenceKind(record.Spec.scopedResource()))
}

func TestDecodeRoleBinding(t *testing.T) {
	const payload = `{
      "api_version": "iam/v2",
      "kind": "RoleBinding",
      "id": "rb-abc123",
      "metadata": {"self": "https://api.confluent.cloud/iam/v2/role-bindings/rb-abc123", "resource_name": "crn://..."},
      "principal": "User:sa-abc123",
      "role_name": "CloudClusterAdmin",
      "crn_pattern": "crn://confluent.cloud/organization=o-1/environment=env-abc123/cloud-cluster=lkc-abc123"
    }`

	var record roleBindingRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	assert.Equal(t, "rb-abc123", record.ID)
	assert.Equal(t, "User:sa-abc123", record.Principal)
	assert.Equal(t, "CloudClusterAdmin", record.RoleName)
	assert.Equal(t, "crn://confluent.cloud/organization=o-1/environment=env-abc123/cloud-cluster=lkc-abc123", record.CrnPattern)
}

func TestDecodeServiceAccountAndUser(t *testing.T) {
	var account serviceAccountRecord
	require.NoError(t, json.Unmarshal([]byte(`{
      "id": "sa-abc123",
      "metadata": {"self": "https://...", "resource_name": "crn://confluent.cloud/service-account=sa-abc123", "created_at": "2020-01-02T03:04:05Z"},
      "display_name": "payments-producer",
      "description": "writes to payments"
    }`), &account))
	assert.Equal(t, "sa-abc123", account.ID)
	assert.Equal(t, "payments-producer", account.DisplayName)
	assert.Equal(t, "writes to payments", account.Description)
	require.NotNil(t, account.Metadata.CreatedAt.Time())

	var user userRecord
	require.NoError(t, json.Unmarshal([]byte(`{
      "id": "u-abc123",
      "metadata": {"self": "https://...", "resource_name": "crn://confluent.cloud/user=u-abc123"},
      "email": "marty@example.com",
      "full_name": "Marty McFly",
      "auth_type": "AUTH_TYPE_LOCAL"
    }`), &user))
	assert.Equal(t, "u-abc123", user.ID)
	assert.Equal(t, "marty@example.com", user.Email)
	assert.Equal(t, "Marty McFly", user.FullName)
	assert.Equal(t, "AUTH_TYPE_LOCAL", user.AuthType)
}

func TestDecodeSchemaRegistryCluster(t *testing.T) {
	const payload = `{
      "api_version": "srcm/v3",
      "kind": "Cluster",
      "id": "lsrc-abc123",
      "metadata": {"self": "https://...", "resource_name": "crn://..."},
      "spec": {
        "display_name": "Stream Governance Package",
        "package": "ADVANCED",
        "http_endpoint": "https://psrc-1.us-east-2.aws.confluent.cloud",
        "catalog_http_endpoint": "https://psrc-1.us-east-2.aws.confluent.cloud/catalog",
        "private_http_endpoint": "https://lsrc-abc123.us-east-2.aws.private.confluent.cloud",
        "private_networking_config": {
          "regional_endpoints": {"us-east-2": "https://lsrc-abc123.us-east-2.aws.private.confluent.cloud"}
        },
        "cloud": "AWS",
        "region": "us-east-2",
        "environment": {"id": "env-abc123", "related": "https://...", "resource_name": "crn://..."}
      },
      "status": {"phase": "PROVISIONED"}
    }`

	var record schemaRegistryRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	require.NotNil(t, record.Spec)
	assert.Equal(t, "ADVANCED", record.Spec.Package)
	assert.Equal(t, "https://psrc-1.us-east-2.aws.confluent.cloud", record.Spec.HTTPEndpoint)
	assert.Equal(t, "https://psrc-1.us-east-2.aws.confluent.cloud/catalog", record.Spec.CatalogHTTPEndpoint)
	assert.Equal(t, "https://lsrc-abc123.us-east-2.aws.private.confluent.cloud", record.Spec.PrivateHTTPEndpoint)
	require.NotNil(t, record.Spec.PrivateNetworkingConfig)
	assert.Equal(t, map[string]string{"us-east-2": "https://lsrc-abc123.us-east-2.aws.private.confluent.cloud"},
		record.Spec.PrivateNetworkingConfig.RegionalEndpoints)
	assert.Equal(t, "env-abc123", refID(record.Spec.Environment))
	require.NotNil(t, record.Status)
	assert.Equal(t, "PROVISIONED", record.Status.Phase)
}

func TestDecodeNetwork(t *testing.T) {
	const payload = `{
      "api_version": "networking/v1",
      "kind": "Network",
      "id": "n-abc123",
      "metadata": {"self": "https://...", "resource_name": "crn://...", "created_at": "2024-04-04T00:00:00Z"},
      "spec": {
        "display_name": "prod-aws-us-east1",
        "cloud": "AWS",
        "region": "us-east-1",
        "connection_types": ["PRIVATELINK"],
        "cidr": "10.200.0.0/16",
        "zones": ["use1-az1", "use1-az2", "use1-az3"],
        "dns_config": {"resolution": "PRIVATE"},
        "environment": {"id": "env-abc123", "related": "https://...", "resource_name": "crn://..."}
      },
      "status": {
        "phase": "READY",
        "supported_connection_types": ["PRIVATELINK"],
        "active_connection_types": ["PRIVATELINK"],
        "dns_domain": "abc123.us-east-1.aws.glb.confluent.cloud"
      }
    }`

	var record networkRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &record))
	require.NotNil(t, record.Spec)
	assert.Equal(t, "prod-aws-us-east1", record.Spec.DisplayName)
	assert.Equal(t, "10.200.0.0/16", record.Spec.Cidr)
	assert.Equal(t, []string{"PRIVATELINK"}, record.Spec.ConnectionTypes)
	assert.Equal(t, []string{"use1-az1", "use1-az2", "use1-az3"}, record.Spec.Zones)
	require.NotNil(t, record.Spec.DnsConfig)
	assert.Equal(t, "PRIVATE", record.Spec.DnsConfig.Resolution)
	assert.Equal(t, "env-abc123", refID(record.Spec.Environment))
	require.NotNil(t, record.Status)
	assert.Equal(t, "READY", record.Status.Phase)
	assert.Equal(t, []string{"PRIVATELINK"}, record.Status.ActiveConnectionTypes)
	assert.Equal(t, "abc123.us-east-1.aws.glb.confluent.cloud", record.Status.DnsDomain)
}

func TestDecodeEncryptionKeys(t *testing.T) {
	t.Run("aws", func(t *testing.T) {
		const payload = `{
          "id": "cck-aws",
          "metadata": {"self": "https://...", "resource_name": "crn://...", "created_at": "2024-01-01T00:00:00Z"},
          "key": {
            "kind": "AwsKey",
            "key_arn": "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab",
            "roles": ["arn:aws:iam::123456789876:role/block_storage_manager"]
          },
          "display_name": "billing key",
          "provider": "AWS",
          "state": "IN_USE",
          "validation": {"phase": "VALID", "message": "", "since": "2024-03-20T15:30:00Z", "region": "us-west-2"}
        }`

		var record encryptionKeyRecord
		require.NoError(t, json.Unmarshal([]byte(payload), &record))
		require.NotNil(t, record.Key)
		assert.Equal(t, "AwsKey", record.Key.Kind)
		assert.Equal(t, "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab", keyReferenceOf(record.Key))
		assert.Equal(t, []string{"arn:aws:iam::123456789876:role/block_storage_manager"}, record.Key.Roles)
		assert.Equal(t, "AWS", record.Provider)
		assert.Equal(t, "IN_USE", record.State)
		require.NotNil(t, record.Validation)
		assert.Equal(t, "VALID", record.Validation.Phase)
		require.NotNil(t, record.Validation.Since.Time())
	})

	t.Run("azure", func(t *testing.T) {
		const payload = `{
          "id": "cck-azure",
          "key": {
            "kind": "AzureKey",
            "key_id": "https://vault-name.vault.azure.net/keys/key-name",
            "key_vault_id": "/subscriptions/0000/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/vault-name",
            "tenant_id": "00000000-0000-0000-0000-000000000000",
            "application_id": "app-1"
          },
          "provider": "Azure",
          "state": "AVAILABLE",
          "validation": {"phase": "INVALID", "message": "Access to key denied.", "since": "2024-03-20T15:30:00Z"}
        }`

		var record encryptionKeyRecord
		require.NoError(t, json.Unmarshal([]byte(payload), &record))
		assert.Equal(t, "https://vault-name.vault.azure.net/keys/key-name", keyReferenceOf(record.Key))
		assert.Equal(t, "/subscriptions/0000/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/vault-name", record.Key.KeyVaultID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000000", record.Key.TenantID)
		assert.Equal(t, "app-1", record.Key.ApplicationID)
		assert.Equal(t, "INVALID", record.Validation.Phase)
		assert.Equal(t, "Access to key denied.", record.Validation.Message)
	})

	t.Run("gcp", func(t *testing.T) {
		const payload = `{
          "id": "cck-gcp",
          "key": {
            "kind": "GcpKey",
            "key_id": "projects/p/locations/us-central1/keyRings/kr/cryptoKeys/k/cryptoKeyVersions/3",
            "security_group": "testgroupid@domain.com"
          },
          "provider": "GCP",
          "state": "AVAILABLE"
        }`

		var record encryptionKeyRecord
		require.NoError(t, json.Unmarshal([]byte(payload), &record))
		assert.Equal(t, "projects/p/locations/us-central1/keyRings/kr/cryptoKeys/k/cryptoKeyVersions/3", keyReferenceOf(record.Key))
		assert.Equal(t, "testgroupid@domain.com", record.Key.SecurityGroup)
		assert.Nil(t, record.Validation, "a key with no validation block must not report one")
	})
}

func TestDecodeAuditLogPayload(t *testing.T) {
	t.Run("organization at the top level", func(t *testing.T) {
		const payload = `{
          "organization": {
            "id": 42,
            "name": "Acme",
            "audit_log": {
              "cluster_id": "lkc-audit",
              "account_id": "env-audit",
              "topic_name": "confluent-audit-log-events",
              "service_account_id": 98765,
              "service_account_resource_id": "sa-audit"
            }
          }
        }`

		var payloadRecord accountPayload
		require.NoError(t, json.Unmarshal([]byte(payload), &payloadRecord))
		record := payloadRecord.auditLog()
		require.NotNil(t, record)
		assert.Equal(t, "lkc-audit", record.ClusterID)
		assert.Equal(t, "env-audit", record.AccountID)
		assert.Equal(t, "confluent-audit-log-events", record.TopicName)
		assert.Equal(t, "sa-audit", record.ServiceAccountResourceID)
		enabled, known := auditLogEnabled(record)
		assert.True(t, known)
		assert.True(t, enabled)
	})

	t.Run("organization nested under the user", func(t *testing.T) {
		const payload = `{
          "user": {"organization": {"audit_log": {"cluster_id": "lkc-audit", "topic_name": "t", "service_account_id": 1}}}
        }`

		var payloadRecord accountPayload
		require.NoError(t, json.Unmarshal([]byte(payload), &payloadRecord))
		record := payloadRecord.auditLog()
		require.NotNil(t, record)
		assert.Equal(t, "lkc-audit", record.ClusterID)
		enabled, known := auditLogEnabled(record)
		assert.True(t, known)
		assert.True(t, enabled)
	})

	// A payload with no audit log block has not answered the question. It must
	// not read as "auditing is off", which would be a clean pass on an
	// organization that may well be audited.
	t.Run("an organization with no audit log block is unknown, not disabled", func(t *testing.T) {
		var payloadRecord accountPayload
		require.NoError(t, json.Unmarshal([]byte(`{"organization":{"id":1,"name":"Acme"}}`), &payloadRecord))
		assert.Nil(t, payloadRecord.auditLog())

		enabled, known := auditLogEnabled(payloadRecord.auditLog())
		assert.False(t, known, "an absent block must report the answer as unknown")
		assert.False(t, enabled, "the value is meaningless while known is false")
	})

	// An empty answer from the endpoint is the same absence one step removed.
	t.Run("an empty payload is unknown, not disabled", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"organization":null}`, `{"user":{"organization":{}}}`} {
			var payloadRecord accountPayload
			require.NoError(t, json.Unmarshal([]byte(body), &payloadRecord), body)
			_, known := auditLogEnabled(payloadRecord.auditLog())
			assert.False(t, known, body)
		}
	})

	// The API affirmatively saying auditing is off is the only thing that may
	// read as false.
	t.Run("an audit log block with no writing account is an answered false", func(t *testing.T) {
		var payloadRecord accountPayload
		require.NoError(t, json.Unmarshal([]byte(`{"organization":{"audit_log":{"service_account_id":0}}}`), &payloadRecord))
		record := payloadRecord.auditLog()
		require.NotNil(t, record)

		enabled, known := auditLogEnabled(record)
		assert.True(t, known, "a present block is an answer")
		assert.False(t, enabled)
	})

	t.Run("either half of the writing account counts as enabled", func(t *testing.T) {
		numericOnly := &auditLogRecord{ServiceAccountID: 7}
		enabled, known := auditLogEnabled(numericOnly)
		assert.True(t, known)
		assert.True(t, enabled)

		resourceOnly := &auditLogRecord{ServiceAccountResourceID: "sa-1"}
		enabled, known = auditLogEnabled(resourceOnly)
		assert.True(t, known)
		assert.True(t, enabled)
	})
}

func TestConfluentTimeDecoding(t *testing.T) {
	type holder struct {
		At confluentTime `json:"at"`
	}

	t.Run("RFC 3339 with an offset", func(t *testing.T) {
		var h holder
		require.NoError(t, json.Unmarshal([]byte(`{"at":"2006-01-02T15:04:05-07:00"}`), &h))
		require.NotNil(t, h.At.Time())
		assert.Equal(t, 2006, h.At.Time().Year())
		assert.True(t, h.At.Time().Equal(time.Date(2006, 1, 2, 22, 4, 5, 0, time.UTC)))
	})

	// An absent timestamp must stay null. Decoding it to the zero time would
	// report 1 January year 1 as a real date, and an age computed from it as
	// two millennia.
	t.Run("absent, null and empty stay null", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"at":null}`, `{"at":""}`} {
			var h holder
			require.NoError(t, json.Unmarshal([]byte(body), &h), body)
			assert.Nil(t, h.At.Time(), body)
		}
	})

	// A timestamp whose shape changed is reported as null rather than failing
	// the whole record, so one field cannot blind an entire listing.
	t.Run("an unparseable timestamp is null, not an error", func(t *testing.T) {
		var h holder
		require.NoError(t, json.Unmarshal([]byte(`{"at":"yesterday"}`), &h))
		assert.Nil(t, h.At.Time())
	})

	t.Run("a non-string timestamp is an error", func(t *testing.T) {
		var h holder
		require.Error(t, json.Unmarshal([]byte(`{"at":12345}`), &h))
	})
}
