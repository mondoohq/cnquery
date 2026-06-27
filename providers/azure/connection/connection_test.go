// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestSelectAzureCredential_WorkloadIdentity(t *testing.T) {
	// Set up an env var pointing to a (non-existent) token file.
	// Construction of WorkloadIdentityCredential does not read the file —
	// it is read lazily at GetToken time — so the file need not exist.
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id":       "tid",
			"client-id":       "cid",
			"subscription-id": "sub",
		},
		Credentials: nil,
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
}

func TestSelectAzureCredential_WorkloadIdentity_ViaOption(t *testing.T) {
	// Ensure env var is not set so we test the option path exclusively.
	// Save and restore to avoid permanently clobbering the env for later tests.
	prev, ok := os.LookupEnv("AZURE_FEDERATED_TOKEN_FILE")
	os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
	t.Cleanup(func() {
		if ok {
			os.Setenv("AZURE_FEDERATED_TOKEN_FILE", prev)
		} else {
			os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
		}
	})

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id":                  "tid",
			"client-id":                  "cid",
			"subscription-id":            "sub",
			"azure-federated-token-file": "/tmp/x.jwt",
		},
		Credentials: nil,
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
}

// TestSelectAzureCredential_VaultCredWins asserts that an explicit vault
// credential takes precedence over a federated token file env var. Even when
// AZURE_FEDERATED_TOKEN_FILE is set, the vault credential path must be taken,
// so the returned credential must not be a WorkloadIdentityCredential.
func TestSelectAzureCredential_VaultCredWins(t *testing.T) {
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id": "tid",
			"client-id": "cid",
		},
		Credentials: []*vault.Credential{
			{Type: vault.CredentialType_password, Secret: []byte("secret")},
		},
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
	_, isWIF := cred.(*azidentity.WorkloadIdentityCredential)
	require.False(t, isWIF, "vault credential must win over federated token file env var")
}

// TestSelectAzureCredential_DefaultChain asserts that when neither a vault
// credential nor a federated token file is present, selectAzureCredential
// falls through to the default credential chain — which must not be a
// WorkloadIdentityCredential.
func TestSelectAzureCredential_DefaultChain(t *testing.T) {
	// Unset the env var and restore it after the test.
	prev, ok := os.LookupEnv("AZURE_FEDERATED_TOKEN_FILE")
	os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
	t.Cleanup(func() {
		if ok {
			os.Setenv("AZURE_FEDERATED_TOKEN_FILE", prev)
		} else {
			os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
		}
	})

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id": "tid",
			"client-id": "cid",
		},
		Credentials: nil,
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
	_, isWIF := cred.(*azidentity.WorkloadIdentityCredential)
	require.False(t, isWIF, "no vault cred + no token file must fall through to the default chain, not WorkloadIdentityCredential")
}
