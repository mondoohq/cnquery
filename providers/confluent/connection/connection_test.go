// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestCredentialsFromConf(t *testing.T) {
	t.Run("bare password is the cloud API secret", func(t *testing.T) {
		cloud, kafka := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("cloud")},
			},
		})
		assert.Equal(t, "cloud", cloud)
		assert.Empty(t, kafka)
	})

	t.Run("the tagged credential is the kafka secret", func(t *testing.T) {
		cloud, kafka := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, User: "kafka-api-secret", Secret: []byte("kafka")},
			},
		})
		assert.Empty(t, cloud)
		assert.Equal(t, "kafka", kafka)
	})

	t.Run("both can be present at once", func(t *testing.T) {
		cloud, kafka := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("cloud")},
				{Type: mondoovault.CredentialType_password, User: "kafka-api-secret", Secret: []byte("kafka")},
			},
		})
		assert.Equal(t, "cloud", cloud)
		assert.Equal(t, "kafka", kafka)
	})

	t.Run("empty secrets and wrong types are ignored", func(t *testing.T) {
		cloud, kafka := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				nil,
				{Type: mondoovault.CredentialType_password, Secret: []byte{}},
				{Type: mondoovault.CredentialType_private_key, Secret: []byte("key")},
			},
		})
		assert.Empty(t, cloud)
		assert.Empty(t, kafka)
	})

	t.Run("nil config", func(t *testing.T) {
		cloud, kafka := credentialsFromConf(nil)
		assert.Empty(t, cloud)
		assert.Empty(t, kafka)
	})
}

func TestNewConfluentConnectionRequiresBothHalvesOfTheKey(t *testing.T) {
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvAPISecret, "")

	t.Run("no credentials at all", func(t *testing.T) {
		_, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), EnvAPIKey)
	})

	t.Run("a key without a secret is rejected", func(t *testing.T) {
		_, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
			Options: map[string]string{OptionAPIKey: "key"},
		})
		require.Error(t, err)
	})

	t.Run("environment supplies both halves", func(t *testing.T) {
		t.Setenv(EnvAPIKey, "env-key")
		t.Setenv(EnvAPISecret, "env-secret")

		conn, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{})
		require.NoError(t, err)
		assert.Equal(t, DefaultAPIBaseURL, conn.BaseURL())
	})

	t.Run("the base URL override drops a trailing slash", func(t *testing.T) {
		conn, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
			Options: map[string]string{
				OptionAPIKey:  "key",
				OptionBaseURL: "https://api.example.com/",
			},
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("secret")},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com", conn.BaseURL())
	})
}

func TestKafkaEnvSuffix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"lkc-abc123", "LKC_ABC123"},
		{"lkc-ABC123", "LKC_ABC123"},
		{"lkc_abc", "LKC_ABC"},
		{"", ""},
		// a leading digit is not a legal environment variable name
		{"1kc-abc", ""},
		// anything that cannot appear in a variable name disables the
		// per-cluster lookup rather than silently reading another variable
		{"lkc.abc", ""},
		{"lkc abc", ""},
		{"lkc/abc", ""},
		{"lkc$abc", ""},
		{"-", ""},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, KafkaEnvSuffix(tc.in))
		})
	}
}

func TestKafkaCredentialsFor(t *testing.T) {
	newConn := func(t *testing.T) *ConfluentConnection {
		t.Helper()
		conn, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
			Options: map[string]string{OptionAPIKey: "key"},
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("secret")},
			},
		})
		require.NoError(t, err)
		return conn
	}

	t.Run("no kafka key configured", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "")
		t.Setenv(EnvKafkaAPISecret, "")
		conn := newConn(t)
		key, secret := conn.KafkaCredentialsFor("lkc-abc123")
		assert.Empty(t, key)
		assert.Empty(t, secret)
	})

	t.Run("the connection-wide pair applies to every cluster", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "wide-key")
		t.Setenv(EnvKafkaAPISecret, "wide-secret")
		conn := newConn(t)
		key, secret := conn.KafkaCredentialsFor("lkc-abc123")
		assert.Equal(t, "wide-key", key)
		assert.Equal(t, "wide-secret", secret)
	})

	t.Run("a per-cluster pair wins", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "wide-key")
		t.Setenv(EnvKafkaAPISecret, "wide-secret")
		t.Setenv(EnvKafkaAPIKey+"_LKC_ABC123", "scoped-key")
		t.Setenv(EnvKafkaAPISecret+"_LKC_ABC123", "scoped-secret")

		conn := newConn(t)
		key, secret := conn.KafkaCredentialsFor("lkc-abc123")
		assert.Equal(t, "scoped-key", key)
		assert.Equal(t, "scoped-secret", secret)

		// a cluster without its own pair still gets the connection-wide one
		key, secret = conn.KafkaCredentialsFor("lkc-other")
		assert.Equal(t, "wide-key", key)
		assert.Equal(t, "wide-secret", secret)
	})

	// Half a per-cluster pair is not a usable credential, and must not shadow
	// the complete connection-wide one.
	t.Run("an incomplete per-cluster pair falls back", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "wide-key")
		t.Setenv(EnvKafkaAPISecret, "wide-secret")
		t.Setenv(EnvKafkaAPIKey+"_LKC_ABC123", "scoped-key")
		t.Setenv(EnvKafkaAPISecret+"_LKC_ABC123", "")

		conn := newConn(t)
		key, secret := conn.KafkaCredentialsFor("lkc-abc123")
		assert.Equal(t, "wide-key", key)
		assert.Equal(t, "wide-secret", secret)
	})
}

func TestKafkaTarget(t *testing.T) {
	newConn := func(t *testing.T) *ConfluentConnection {
		t.Helper()
		conn, err := NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
			Options: map[string]string{OptionAPIKey: "key"},
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("secret")},
			},
		})
		require.NoError(t, err)
		return conn
	}

	// Without a cluster key the topic and ACL listings must fail loudly. An
	// empty list would read as a cluster with no ACLs at all, which is the
	// answer an audit is looking for.
	t.Run("a missing kafka key is an error, not an empty result", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "")
		t.Setenv(EnvKafkaAPISecret, "")

		conn := newConn(t)
		_, err := conn.KafkaTarget("lkc-abc123", "https://pkc-1.us-east-1.aws.confluent.cloud:443")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lkc-abc123")
		assert.Contains(t, err.Error(), EnvKafkaAPIKey+"_LKC_ABC123")
	})

	t.Run("a cluster with no REST endpoint is an error", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "k")
		t.Setenv(EnvKafkaAPISecret, "s")

		conn := newConn(t)
		_, err := conn.KafkaTarget("lkc-abc123", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REST endpoint")
	})

	t.Run("a usable target trims the endpoint and carries the cluster key", func(t *testing.T) {
		t.Setenv(EnvKafkaAPIKey, "k")
		t.Setenv(EnvKafkaAPISecret, "s")

		conn := newConn(t)
		target, err := conn.KafkaTarget("lkc-abc123", "https://pkc-1.us-east-1.aws.confluent.cloud:443/")
		require.NoError(t, err)
		assert.Equal(t, "https://pkc-1.us-east-1.aws.confluent.cloud:443", target.BaseURL)
		assert.Equal(t, "k", target.Key)
		assert.Equal(t, "s", target.Secret)
		assert.False(t, target.SupportsPageSize)
	})
}

func TestOrganizationFromCRN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "environment CRN",
			in:   "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-abc123",
			want: "9bb441c4-edef-46ac-8a41-c49e44a3fd9a",
		},
		{
			name: "cluster CRN",
			in:   "crn://confluent.cloud/organization=o-123/environment=env-1/cloud-cluster=lkc-1",
			want: "o-123",
		},
		{
			name: "no organization segment",
			in:   "crn://confluent.cloud/kafka=lkc-f3a90de",
			want: "",
		},
		{name: "empty", in: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OrganizationFromCRN(tc.in))
		})
	}
}

func TestNewConfluentOrgIdentifier(t *testing.T) {
	assert.Equal(t,
		PlatformIdConfluentOrg+"9bb441c4-edef-46ac-8a41-c49e44a3fd9a",
		NewConfluentOrgIdentifier("9bb441c4-edef-46ac-8a41-c49e44a3fd9a"))

	// two organizations must not collide
	assert.NotEqual(t, NewConfluentOrgIdentifier("o-a"), NewConfluentOrgIdentifier("o-b"))
}

func TestNewConfluentOrgPlatform(t *testing.T) {
	pf := NewConfluentOrgPlatform("o-a")
	assert.Equal(t, []string{"saas", "confluent", "organization", "o-a"}, pf.TechnologyUrlSegments)
	assert.Equal(t, "confluent", pf.Name)
	assert.Equal(t, "api", pf.Kind)
}
