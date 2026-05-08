// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLikelySecretName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"AzureWebJobsStorage", false},
		{"FUNCTIONS_WORKER_RUNTIME", false},
		{"WEBSITE_RUN_FROM_PACKAGE", false},
		{"DB_PASSWORD", true},
		{"db_password", true},
		{"AzureWebJobsStorageKey", true},
		{"StorageAccountKey", true},
		{"ApiSecret", true},
		{"my_token", true},
		{"OAUTH_TOKEN", true},
		// Bare "connection" should NOT be flagged — the pattern only fires
		// on connection-string indicators ("connectionString", "connStr", etc).
		{"DATABASE_CONNECTION", false},
		{"connectionRetryCount", false},
		{"connectionTimeout", false},
		{"DefaultConnectionString", true},
		{"AZURE_STORAGE_CONNECTION_STRING", true},
		{"ConnStr", true},
		{"ConnectionStrings:Default", true},
		{"WEBSITE_INSTANCE_ID", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isLikelySecretName(tc.name))
		})
	}
}

func TestHasKeyVaultRefValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty", "", false},
		{"plaintext", "abc123", false},
		{"plain connection string", "Server=tcp:foo.database.windows.net", false},
		{"key vault ref by secret uri", "@Microsoft.KeyVault(SecretUri=https://kv.vault.azure.net/secrets/x/abc)", true},
		{"key vault ref by name+vault", "@Microsoft.KeyVault(VaultName=kv;SecretName=mySecret)", true},
		{"with leading whitespace", "  @Microsoft.KeyVault(SecretUri=...)", true},
		{"key vault wrong prefix not matched", "Microsoft.KeyVault(SecretUri=...)", false},
		{"contains but doesn't start with", "fallback=@Microsoft.KeyVault(...)", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasKeyVaultRefValue(tc.value))
		})
	}
}

func TestSortedSettingKeys(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		assert.Empty(t, sortedSettingKeys(nil))
	})
	t.Run("alphabetical order", func(t *testing.T) {
		v1 := "x"
		v2 := "y"
		v3 := "z"
		input := map[string]*string{
			"zeta":  &v3,
			"alpha": &v1,
			"mu":    &v2,
		}
		assert.Equal(t, []string{"alpha", "mu", "zeta"}, sortedSettingKeys(input))
	})
}
