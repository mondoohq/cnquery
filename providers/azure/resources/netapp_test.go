// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/netapp/armnetapp/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The regression this guards: NetApp reports the vault URI and the key name as
// two fields, so the key identifier has to be assembled. Getting the join wrong
// yields a URI that parseKeyVaultId rejects, and the encryption key of a
// customer-managed-key account reads as unresolvable.
func TestNetAppKeyVaultKeyId(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("vault URI and key name are joined into a versionless key id", func(t *testing.T) {
		assert.Equal(t,
			"https://contoso.vault.azure.net/keys/netapp-cmk",
			netAppKeyVaultKeyId(str("https://contoso.vault.azure.net"), str("netapp-cmk")))
	})

	t.Run("a trailing slash on the vault URI does not double up", func(t *testing.T) {
		assert.Equal(t,
			"https://contoso.vault.azure.net/keys/netapp-cmk",
			netAppKeyVaultKeyId(str("https://contoso.vault.azure.net/"), str("netapp-cmk")))
	})

	// A platform-key account reports no vault at all. Assembling a key id out of
	// the empty halves would produce a reference that cannot resolve, where the
	// truthful answer is that there is no customer-managed key.
	t.Run("no key id without both halves", func(t *testing.T) {
		assert.Equal(t, "", netAppKeyVaultKeyId(nil, nil))
		assert.Equal(t, "", netAppKeyVaultKeyId(str("https://contoso.vault.azure.net"), nil))
		assert.Equal(t, "", netAppKeyVaultKeyId(nil, str("netapp-cmk")))
		assert.Equal(t, "", netAppKeyVaultKeyId(str(""), str("netapp-cmk")))
		assert.Equal(t, "", netAppKeyVaultKeyId(str("https://contoso.vault.azure.net"), str("")))
	})

	t.Run("the assembled id parses as a key vault identifier", func(t *testing.T) {
		parsed, err := parseKeyVaultId(netAppKeyVaultKeyId(
			str("https://contoso.vault.azure.net"), str("netapp-cmk")))
		require.NoError(t, err)
		assert.Equal(t, "contoso", parsed.Vault)
		assert.Equal(t, "keys", parsed.Type)
		assert.Equal(t, "netapp-cmk", parsed.Name)
		assert.Equal(t, "", parsed.Version)
	})
}

func TestNetAppMountTargetIps(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("no mount targets is an empty list, never nil", func(t *testing.T) {
		res := netAppMountTargetIps(nil)
		require.NotNil(t, res)
		assert.Empty(t, res)
	})

	t.Run("nil entries and empty addresses are skipped", func(t *testing.T) {
		res := netAppMountTargetIps([]*armnetapp.MountTargetProperties{
			nil,
			{},
			{IPAddress: str("")},
			{IPAddress: str("10.0.1.4")},
		})
		assert.Equal(t, []any{"10.0.1.4"}, res)
	})

	t.Run("every address of a multi-target volume is reported", func(t *testing.T) {
		res := netAppMountTargetIps([]*armnetapp.MountTargetProperties{
			{IPAddress: str("10.0.1.4")},
			{IPAddress: str("10.0.2.4")},
		})
		assert.Equal(t, []any{"10.0.1.4", "10.0.2.4"}, res)
	})
}
