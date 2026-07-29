// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// resetCredentialCache drops every cached credential. Tests need it because the
// cache outlives any one of them.
func resetCredentialCache() {
	credentialMu.Lock()
	defer credentialMu.Unlock()
	clear(credentialCache)
}

// The credential is built once per identity and shared: a scan opens a
// connection per asset, and one credential per connection is a token request per
// asset for a tenant that already had a token.
func TestSelectMs365Credential_CachesAcrossConnections(t *testing.T) {
	resetCredentialCache()
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	first, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tid", OptionClientID: "cid"},
	})
	require.NoError(t, err)

	second, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tid", OptionClientID: "cid", OptionOrganization: "org"},
	})
	require.NoError(t, err)
	require.Same(t, first, second, "a connection under the same identity must reuse the credential")
}

// Handing tenant A's token to tenant B would not fail cleanly -- it would read
// the wrong tenant -- so each identity gets its own entry, and each is reused.
func TestSelectMs365Credential_CachesPerIdentity(t *testing.T) {
	resetCredentialCache()
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	first, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tenant-a", OptionClientID: "cid"},
	})
	require.NoError(t, err)

	second, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tenant-b", OptionClientID: "cid"},
	})
	require.NoError(t, err)
	require.NotSame(t, first, second, "a different tenant must get its own credential")

	againA, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tenant-a", OptionClientID: "cid"},
	})
	require.NoError(t, err)
	require.Same(t, first, againA)
}

// An unusable auth-method belongs to the connection that named it. Consulting
// the cache first would let a typo through on every connection after the one
// that built the credential.
func TestSelectMs365Credential_RejectsBadAuthMethodEvenWhenCached(t *testing.T) {
	resetCredentialCache()
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	_, err := selectMs365Credential(&inventory.Config{
		Options: map[string]string{OptionTenantID: "tid", OptionClientID: "cid"},
	})
	require.NoError(t, err)

	_, err = selectMs365Credential(&inventory.Config{
		Options: map[string]string{
			OptionTenantID:   "tid",
			OptionClientID:   "cid",
			OptionAuthMethod: "service-principal",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "service-principal")
}
