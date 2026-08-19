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

func passwordConf(secret string, options map[string]string) *inventory.Config {
	conf := &inventory.Config{Type: "newrelic", Options: options}
	if secret != "" {
		conf.Credentials = append(conf.Credentials, &mondoovault.Credential{
			Type:   mondoovault.CredentialType_password,
			Secret: []byte(secret),
		})
	}
	return conf
}

func TestNewConnectionRequiresCredentialsAndAccount(t *testing.T) {
	t.Setenv("NEW_RELIC_API_KEY", "")
	t.Setenv("NEW_RELIC_ACCOUNT_ID", "")
	t.Setenv("NEW_RELIC_REGION", "")

	t.Run("no key", func(t *testing.T) {
		_, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("", map[string]string{OptionAccountID: "1234567"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user API key")
	})

	t.Run("no account", func(t *testing.T) {
		_, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("NRAK-X", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account ID")
	})

	// A non-numeric account ID has to be refused here. Passed through to
	// NerdGraph it would fail deep inside one query and read as an account with
	// no keys and no alert policies.
	t.Run("non-numeric account", func(t *testing.T) {
		_, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("NRAK-X", map[string]string{OptionAccountID: "my-account"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be numeric")
	})

	t.Run("zero account", func(t *testing.T) {
		_, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("NRAK-X", map[string]string{OptionAccountID: "0"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive number")
	})

	t.Run("unknown region", func(t *testing.T) {
		_, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("NRAK-X", map[string]string{
			OptionAccountID: "1234567",
			OptionRegion:    "apac",
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown New Relic region")
	})
}

func TestNewConnectionUsesOptionsAndEnvironment(t *testing.T) {
	t.Setenv("NEW_RELIC_API_KEY", "NRAK-FROM-ENV")
	t.Setenv("NEW_RELIC_ACCOUNT_ID", "7654321")
	t.Setenv("NEW_RELIC_REGION", "eu")

	t.Run("environment only", func(t *testing.T) {
		conn, err := NewNewrelicConnection(1, &inventory.Asset{}, &inventory.Config{Type: "newrelic"})
		require.NoError(t, err)
		assert.Equal(t, 7654321, conn.AccountID())
		assert.Equal(t, RegionEU, conn.Region())
		assert.Equal(t, endpointEU, conn.Client().Endpoint())
	})

	// Explicit options win over the environment, so a shell that happens to
	// carry another account's variables cannot silently redirect a scan.
	t.Run("options win", func(t *testing.T) {
		conn, err := NewNewrelicConnection(1, &inventory.Asset{}, passwordConf("NRAK-FROM-FLAG", map[string]string{
			OptionAccountID: " 1234567 ",
			OptionRegion:    "us",
		}))
		require.NoError(t, err)
		assert.Equal(t, 1234567, conn.AccountID())
		assert.Equal(t, RegionUS, conn.Region())
		assert.Equal(t, endpointUS, conn.Client().Endpoint())
	})
}

func TestCredentialFromConf(t *testing.T) {
	assert.Equal(t, "", credentialFromConf(nil))
	assert.Equal(t, "", credentialFromConf(&inventory.Config{}))
	assert.Equal(t, "", credentialFromConf(&inventory.Config{
		Credentials: []*mondoovault.Credential{nil, {Type: mondoovault.CredentialType_password}},
	}), "an empty secret is not a credential")
	assert.Equal(t, "NRAK-X", credentialFromConf(passwordConf("NRAK-X", nil)))

	// A credential of another kind is not a New Relic key and must not be used
	// as one.
	assert.Equal(t, "", credentialFromConf(&inventory.Config{
		Credentials: []*mondoovault.Credential{{
			Type:   mondoovault.CredentialType_ssh_agent,
			Secret: []byte("not-a-key"),
		}},
	}))
}
