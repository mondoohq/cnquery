// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/registry"
)

// The fixtures under testdata/windows-registry are verbatim output of the two
// PowerShell collection scripts the registrykey resource runs
// (GetRegistryKeyItemScript and GetRegistryKeyChildItemsScript), captured from
// four stock Windows Server hosts over SSH. They carry no host name, no
// machine-unique SID, and no address.
//
// They exist so the pure decode functions behind windows.lsa and
// windows.schannel are exercised against what Windows actually writes, rather
// than against hand-built maps that agree with the implementation by
// construction.
const windowsRegistryFixtureDir = "./testdata/windows-registry"

var windowsFixtureVersions = []string{"ws2016", "ws2019", "ws2022", "ws2025"}

// loadFixtureItems parses a captured registry-value listing and lower-cases the
// value names exactly as readLsaKey and readSchannelKey do.
func loadFixtureItems(t *testing.T, version, name string) map[string]registry.RegistryKeyItem {
	t.Helper()

	f, err := os.Open(filepath.Join(windowsRegistryFixtureDir, version, name+".json"))
	require.NoError(t, err)
	defer f.Close()

	entries, err := registry.ParsePowershellRegistryKeyItems(f)
	require.NoError(t, err)

	res := make(map[string]registry.RegistryKeyItem, len(entries))
	for i := range entries {
		res[strings.ToLower(entries[i].Key)] = entries[i]
	}
	return res
}

func loadFixtureChildren(t *testing.T, version, name string) []registry.RegistryKeyChild {
	t.Helper()

	f, err := os.Open(filepath.Join(windowsRegistryFixtureDir, version, name+".json"))
	require.NoError(t, err)
	defer f.Close()

	children, err := registry.ParsePowershellRegistryKeyChildren(f)
	require.NoError(t, err)
	return children
}

// TestComputeLsaFromLiveCaptures decodes the real Lsa key of each release. The
// values a stock host writes are the same on all four, and the ones it does not
// write must stay null: reporting 0 for an absent LmCompatibilityLevel would
// claim the weakest LAN Manager setting on a host that never configured it.
func TestComputeLsaFromLiveCaptures(t *testing.T) {
	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			items := loadFixtureItems(t, v, "lsa-items")
			lsa := computeLsa(items)

			// values every stock host writes
			require.NotNil(t, lsa.LimitBlankPasswordUse)
			assert.True(t, *lsa.LimitBlankPasswordUse)
			require.NotNil(t, lsa.NoLmHash)
			assert.True(t, *lsa.NoLmHash)
			require.NotNil(t, lsa.RestrictAnonymousSam)
			assert.True(t, *lsa.RestrictAnonymousSam)
			require.NotNil(t, lsa.DisableDomainCreds)
			assert.False(t, *lsa.DisableDomainCreds)
			require.NotNil(t, lsa.EveryoneIncludesAnonymous)
			assert.False(t, *lsa.EveryoneIncludesAnonymous)
			require.NotNil(t, lsa.ForceGuest)
			assert.False(t, *lsa.ForceGuest)
			require.NotNil(t, lsa.RestrictAnonymous)
			assert.Equal(t, int64(0), *lsa.RestrictAnonymous)

			// values a stock host leaves unset: null, never a zero value
			assert.Nil(t, lsa.LmCompatibilityLevel)
			assert.Nil(t, lsa.RestrictRemoteSam)
			assert.Nil(t, lsa.RunAsPpl)
			assert.Nil(t, lsa.SceNoApplyLegacyAuditPolicy)
			assert.Nil(t, lsa.SubmitControl)
			assert.Nil(t, lsa.UseMachineId)
		})
	}
}

// TestComputeLsaNtlmFromLiveCaptures decodes the real MSV1_0, WDigest, and
// pku2u keys. The two minimum-session-security values are the only ones a stock
// host writes; WDigest's UseLogonCredential and pku2u's AllowOnlineID are absent
// everywhere, and absent has to read as null rather than as a configured false.
func TestComputeLsaNtlmFromLiveCaptures(t *testing.T) {
	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			msv10 := loadFixtureItems(t, v, "lsa-msv10-items")
			wdigest := loadFixtureItems(t, v, "wdigest-items")
			// no stock host in this set has an Lsa\pku2u key at all, which the
			// reader turns into an empty map
			pku2u := map[string]registry.RegistryKeyItem{}

			ntlm := computeLsaNtlm(msv10, wdigest, pku2u)

			// 0x20000000 is "require 128-bit encryption"
			require.NotNil(t, ntlm.NtlmMinClientSec)
			assert.Equal(t, int64(0x20000000), *ntlm.NtlmMinClientSec)
			require.NotNil(t, ntlm.NtlmMinServerSec)
			assert.Equal(t, int64(0x20000000), *ntlm.NtlmMinServerSec)

			assert.Nil(t, ntlm.UseLogonCredential)
			assert.Nil(t, ntlm.AllowOnlineId)
			assert.Nil(t, ntlm.AllowNullSessionFallback)
			assert.Nil(t, ntlm.AuditReceivingNtlmTraffic)
			assert.Nil(t, ntlm.RestrictSendingNtlmTraffic)

			// the WDigest key exists and carries other values, so a null
			// UseLogonCredential here is an absent value and not an unread key
			assert.NotEmpty(t, wdigest)
			assert.Contains(t, wdigest, "negotiate")
		})
	}
}

// TestComputeLsaSecureChannelFromLiveCaptures decodes the real
// Services\Netlogon\Parameters key. The four signing and sealing values plus the
// password age are written on every stock host; the GPO-only key that carries
// BlockNetbiosDiscovery does not exist, so that field stays null.
func TestComputeLsaSecureChannelFromLiveCaptures(t *testing.T) {
	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			netlogon := loadFixtureItems(t, v, "netlogon-items")
			// HKLM\SOFTWARE\Policies\Microsoft\Netlogon\Parameters is absent on
			// a host with no GPO applied
			policy := map[string]registry.RegistryKeyItem{}

			sc := computeLsaSecureChannel(netlogon, policy)

			require.NotNil(t, sc.RequireSignOrSeal)
			assert.True(t, *sc.RequireSignOrSeal)
			require.NotNil(t, sc.RequireStrongKey)
			assert.True(t, *sc.RequireStrongKey)
			require.NotNil(t, sc.SealSecureChannel)
			assert.True(t, *sc.SealSecureChannel)
			require.NotNil(t, sc.SignSecureChannel)
			assert.True(t, *sc.SignSecureChannel)
			require.NotNil(t, sc.DisablePasswordChange)
			assert.False(t, *sc.DisablePasswordChange)
			require.NotNil(t, sc.MaximumPasswordAge)
			assert.Equal(t, int64(30), *sc.MaximumPasswordAge)

			assert.Nil(t, sc.BlockNetbiosDiscovery)
			assert.Nil(t, sc.RefusePasswordChange)
			assert.Nil(t, sc.AuditNtlmInDomain)
			assert.Nil(t, sc.VulnerableChannelAllowList)
		})
	}
}

// TestComputeLsaLdapAndFipsFromLiveCaptures decodes the LDAP client key, which
// exists on every host, alongside the NTDS key, which exists only on a domain
// controller. The two server-side fields have to stay null on a member server:
// LDAPServerIntegrity reading 0 would report "unsigned binds accepted" about a
// directory this host does not run.
func TestComputeLsaLdapAndFipsFromLiveCaptures(t *testing.T) {
	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			ldapClient := loadFixtureItems(t, v, "ldap-items")
			// Services\NTDS\Parameters does not exist off a domain controller
			ntds := map[string]registry.RegistryKeyItem{}

			ldap := computeLsaLdap(ntds, ldapClient)
			require.NotNil(t, ldap.ClientIntegrity)
			assert.Equal(t, int64(1), *ldap.ClientIntegrity, "negotiate signing")
			assert.Nil(t, ldap.ServerIntegrity)
			assert.Nil(t, ldap.EnforceChannelBinding)

			fips := computeLsaFips(loadFixtureItems(t, v, "lsa-fips-items"))
			require.NotNil(t, fips)
			assert.False(t, *fips, "FIPS mode is off on a stock host, and explicitly so")
		})
	}
}

// TestSchannelLocalSslStoreFromLiveCaptures reads the two local SSL store
// subkeys of each release. Each names itself in its default value, which is what
// distinguishes the cipher-suite interface from the signature interface: both
// spell their own ordered list `Functions`, so a reader that goes by value name
// alone cannot tell them apart. The supported groups are the `EccCurves` value
// of the cipher-suite subkey.
func TestSchannelLocalSslStoreFromLiveCaptures(t *testing.T) {
	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			cipherIface := loadFixtureItems(t, v, "ssl-00010002-items")
			sigIface := loadFixtureItems(t, v, "ssl-00010003-items")

			assert.Equal(t, "NCRYPT_SCHANNEL_INTERFACE",
				cipherIface["(default)"].Value.String)
			assert.Equal(t, "NCRYPT_SCHANNEL_SIGNATURE_INTERFACE",
				sigIface["(default)"].Value.String)

			// the supported groups, identical on all four releases and equal to
			// what Get-TlsEccCurve reports
			assert.Equal(t, []string{"curve25519", "NistP256", "NistP384"},
				cipherIface["ecccurves"].Value.MultiString)

			// the cipher-suite list is long and version-specific; the
			// signature list is short and made of key/hash pairs. Neither can
			// be mistaken for a group list.
			suites := cipherIface["functions"].Value.MultiString
			assert.Greater(t, len(suites), 20)
			for _, s := range suites {
				assert.True(t, strings.HasPrefix(s, "TLS_"), "unexpected cipher suite %q", s)
			}

			sigs := sigIface["functions"].Value.MultiString
			assert.NotEmpty(t, sigs)
			for _, s := range sigs {
				assert.Contains(t, s, "/", "unexpected signature algorithm %q", s)
				assert.False(t, strings.HasPrefix(s, "TLS_"))
			}
			assert.Contains(t, sigs, "RSA/SHA256")
			assert.Contains(t, sigs, "DSA/SHA1")

			// no stock host in this set enables a post-quantum group
			assert.False(t, pqcKeyExchangeEnabled(cipherIface["ecccurves"].Value.MultiString))
		})
	}
}

// TestSchannelProtocolsFromLiveCaptures reads the real SCHANNEL\Protocols
// listing of each release. Only Windows Server 2016 ships a subkey there, so
// this is also the record of what "unconfigured" looks like: three of the four
// releases enumerate nothing at all, which is why the resource has to report the
// well-known protocol names from its own list rather than from the registry.
func TestSchannelProtocolsFromLiveCaptures(t *testing.T) {
	const protocols = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecurityProviders\SCHANNEL\Protocols`

	expected := map[string][]string{
		"ws2016": {"SSL 2.0"},
		"ws2019": {},
		"ws2022": {},
		"ws2025": {},
	}

	for _, v := range windowsFixtureVersions {
		t.Run(v, func(t *testing.T) {
			children := loadFixtureChildren(t, v, "schannel-protocols-children")
			names := schannelChildNames(protocols, children)
			assert.Equal(t, expected[v], names)

			// whatever the registry enumerates, the reported list is always the
			// full well-known set, so an audit selecting "TLS 1.0" matches
			// something on a host that never configured it
			merged := mergeSchannelNames(schannelWellKnownProtocols, names)
			assert.Equal(t, schannelWellKnownProtocols, merged)
			assert.Contains(t, merged, "TLS 1.3")
			assert.Contains(t, merged, "DTLS 1.2")
		})
	}
}

// TestSchannelConfiguredKeysFromLiveCaptures reads captures taken while the
// SCHANNEL branch was deliberately configured on Windows Server 2016. Two of the
// three cipher names contain a slash, which both the registry path handling and
// the PowerShell child enumeration have to carry through intact: a name that
// fails to round-trip produces no error, it produces a cipher that reads as
// unconfigured, which is indistinguishable from the default state above.
func TestSchannelConfiguredKeysFromLiveCaptures(t *testing.T) {
	const (
		ciphers = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecurityProviders\SCHANNEL\Ciphers`
		kex     = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecurityProviders\SCHANNEL\KeyExchangeAlgorithms`
	)

	cipherNames := schannelChildNames(ciphers,
		loadFixtureChildren(t, "ws2016-configured", "schannel-ciphers-children-configured"))
	assert.Equal(t, []string{"AES 256/256", "RC4 128/128", "Triple DES 168"}, cipherNames)

	// the merge keeps the well-known spelling and order, and adds nothing,
	// because all three names are already well known
	assert.Equal(t, schannelWellKnownCiphers,
		mergeSchannelNames(schannelWellKnownCiphers, cipherNames))

	// the slash-named cipher's own values decode to a real override
	rc4 := loadFixtureItems(t, "ws2016-configured", "schannel-cipher-rc4-128-items")
	assert.True(t, schannelConfigured(rc4, schannelEnabledValue))
	enabled := schannelEnabled(rc4, schannelEnabledValue)
	require.NotNil(t, enabled)
	assert.False(t, *enabled, "RC4 128/128 was set to Enabled=0")

	// a protocol side key: SSL 2.0 on Windows Server 2016 ships with a Client
	// subkey carrying DisabledByDefault and no Enabled value, so the side is
	// configured while its enablement stays null
	ssl20 := loadFixtureItems(t, "ws2016-configured", "schannel-protocol-ssl20-client-items")
	assert.True(t, schannelConfigured(ssl20, schannelEnabledValue, schannelDisabledByDefaultValue))
	assert.Nil(t, schannelEnabled(ssl20, schannelEnabledValue))
	disabledByDefault := schannelEnabled(ssl20, schannelDisabledByDefaultValue)
	require.NotNil(t, disabledByDefault)
	assert.True(t, *disabledByDefault)

	// Diffie-Hellman configured with a minimum key size and no Enabled value:
	// configured has to account for the key-length values or the override is
	// reported as absent
	kexNames := schannelChildNames(kex,
		loadFixtureChildren(t, "ws2016-configured", "schannel-kex-children-configured"))
	assert.Equal(t, []string{"Diffie-Hellman"}, kexNames)
}
