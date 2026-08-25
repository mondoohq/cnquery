// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/registry"
)

// dword builds a registry item map entry for a DWORD value, mimicking how the
// loader lower-cases the value name.
func dword(name string, value int64) (string, registry.RegistryKeyItem) {
	return name, registry.RegistryKeyItem{
		Key:   name,
		Value: registry.RegistryKeyValue{Kind: 4, Number: value},
	}
}

// sz builds a registry item map entry for a REG_SZ value.
func sz(name, value string) (string, registry.RegistryKeyItem) {
	return name, registry.RegistryKeyItem{
		Key:   name,
		Value: registry.RegistryKeyValue{Kind: 1, String: value},
	}
}

func items(pairs ...func() (string, registry.RegistryKeyItem)) map[string]registry.RegistryKeyItem {
	m := map[string]registry.RegistryKeyItem{}
	for _, p := range pairs {
		// keys are lower-cased exactly as readLsaKey does
		k, v := p()
		m[lower(k)] = v
	}
	return m
}

// lower mirrors the strings.ToLower the loader applies; defined locally so the
// test reads naturally.
func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + ('a' - 'A')
		}
	}
	return string(out)
}

func d(name string, value int64) func() (string, registry.RegistryKeyItem) {
	return func() (string, registry.RegistryKeyItem) { return dword(name, value) }
}

func s(name, value string) func() (string, registry.RegistryKeyItem) {
	return func() (string, registry.RegistryKeyItem) { return sz(name, value) }
}

func TestLsaRegIntPtr(t *testing.T) {
	m := items(d("RestrictAnonymous", 1), d("ForceGuest", 0))

	t.Run("present value returns a pointer", func(t *testing.T) {
		p := regIntPtr(m, "RestrictAnonymous")
		require.NotNil(t, p)
		assert.Equal(t, int64(1), *p)
	})

	t.Run("explicit 0 is distinguishable from absent", func(t *testing.T) {
		p := regIntPtr(m, "ForceGuest")
		require.NotNil(t, p)
		assert.Equal(t, int64(0), *p)
	})

	t.Run("absent value returns nil", func(t *testing.T) {
		assert.Nil(t, regIntPtr(m, "DoesNotExist"))
		assert.Nil(t, regIntPtr(map[string]registry.RegistryKeyItem{}, "RestrictAnonymous"))
	})

	t.Run("value name lookup is case insensitive", func(t *testing.T) {
		p := regIntPtr(m, "restrictanonymous")
		require.NotNil(t, p)
		assert.Equal(t, int64(1), *p)
	})
}

func TestLsaRegStringPtr(t *testing.T) {
	m := items(s("restrictremotesam", "O:BAG:BAD:(A;;RC;;;BA)"), s("empty", ""))

	t.Run("present value returns a pointer", func(t *testing.T) {
		p := regStringPtr(m, "restrictremotesam")
		require.NotNil(t, p)
		assert.Equal(t, "O:BAG:BAD:(A;;RC;;;BA)", *p)
	})

	t.Run("explicit empty string is distinguishable from absent", func(t *testing.T) {
		p := regStringPtr(m, "empty")
		require.NotNil(t, p)
		assert.Equal(t, "", *p)
	})

	t.Run("absent value returns nil", func(t *testing.T) {
		assert.Nil(t, regStringPtr(m, "DoesNotExist"))
	})
}

func TestComputeLsa(t *testing.T) {
	t.Run("empty key yields all-null values", func(t *testing.T) {
		v := computeLsa(map[string]registry.RegistryKeyItem{})
		assert.Nil(t, v.DisableDomainCreds)
		assert.Nil(t, v.EveryoneIncludesAnonymous)
		assert.Nil(t, v.ForceGuest)
		assert.Nil(t, v.LimitBlankPasswordUse)
		assert.Nil(t, v.LmCompatibilityLevel)
		assert.Nil(t, v.NoLmHash)
		assert.Nil(t, v.RestrictAnonymous)
		assert.Nil(t, v.RestrictAnonymousSam)
		assert.Nil(t, v.RestrictRemoteSam)
		assert.Nil(t, v.RunAsPpl)
		assert.Nil(t, v.SceNoApplyLegacyAuditPolicy)
		assert.Nil(t, v.SubmitControl)
		assert.Nil(t, v.UseMachineId)
	})

	t.Run("hardened values map every field", func(t *testing.T) {
		m := items(
			d("DisableDomainCreds", 1),
			d("EveryoneIncludesAnonymous", 0),
			d("ForceGuest", 0),
			d("LimitBlankPasswordUse", 1),
			d("LmCompatibilityLevel", 5),
			d("NoLMHash", 1),
			d("RestrictAnonymous", 1),
			d("RestrictAnonymousSAM", 1),
			s("restrictremotesam", "O:BAG:BAD:(A;;RC;;;BA)"),
			d("RunAsPPL", 1),
			d("SCENoApplyLegacyAuditPolicy", 1),
			d("SubmitControl", 0),
			d("UseMachineId", 1),
		)
		v := computeLsa(m)
		require.NotNil(t, v.DisableDomainCreds)
		assert.Equal(t, true, *v.DisableDomainCreds)
		require.NotNil(t, v.EveryoneIncludesAnonymous)
		assert.Equal(t, false, *v.EveryoneIncludesAnonymous)
		require.NotNil(t, v.ForceGuest)
		assert.Equal(t, false, *v.ForceGuest)
		require.NotNil(t, v.LimitBlankPasswordUse)
		assert.Equal(t, true, *v.LimitBlankPasswordUse)
		require.NotNil(t, v.LmCompatibilityLevel)
		assert.Equal(t, int64(5), *v.LmCompatibilityLevel)
		require.NotNil(t, v.NoLmHash)
		assert.Equal(t, true, *v.NoLmHash)
		require.NotNil(t, v.RestrictAnonymous)
		assert.Equal(t, int64(1), *v.RestrictAnonymous)
		require.NotNil(t, v.RestrictAnonymousSam)
		assert.Equal(t, true, *v.RestrictAnonymousSam)
		require.NotNil(t, v.RestrictRemoteSam)
		assert.Equal(t, "O:BAG:BAD:(A;;RC;;;BA)", *v.RestrictRemoteSam)
		require.NotNil(t, v.RunAsPpl)
		assert.Equal(t, int64(1), *v.RunAsPpl)
		require.NotNil(t, v.SceNoApplyLegacyAuditPolicy)
		assert.Equal(t, true, *v.SceNoApplyLegacyAuditPolicy)
		require.NotNil(t, v.SubmitControl)
		assert.Equal(t, false, *v.SubmitControl)
		require.NotNil(t, v.UseMachineId)
		assert.Equal(t, true, *v.UseMachineId)
	})

	t.Run("partial config leaves unset fields null", func(t *testing.T) {
		v := computeLsa(items(d("RestrictAnonymous", 1)))
		require.NotNil(t, v.RestrictAnonymous)
		assert.Equal(t, int64(1), *v.RestrictAnonymous)
		assert.Nil(t, v.ForceGuest)
		assert.Nil(t, v.RestrictRemoteSam)
	})
}

func TestComputeLsaNtlm(t *testing.T) {
	t.Run("empty keys yield all-null values", func(t *testing.T) {
		empty := map[string]registry.RegistryKeyItem{}
		v := computeLsaNtlm(empty, empty, empty)
		assert.Nil(t, v.AllowNullSessionFallback)
		assert.Nil(t, v.AuditReceivingNtlmTraffic)
		assert.Nil(t, v.NtlmMinClientSec)
		assert.Nil(t, v.NtlmMinServerSec)
		assert.Nil(t, v.RestrictSendingNtlmTraffic)
		assert.Nil(t, v.UseLogonCredential)
		assert.Nil(t, v.AllowOnlineId)
	})

	t.Run("reads from the correct source key", func(t *testing.T) {
		msv10 := items(
			d("AllowNullSessionFallback", 0),
			d("AuditReceivingNTLMTraffic", 2),
			d("NTLMMinClientSec", 537395200),
			d("NTLMMinServerSec", 537395200),
			d("RestrictSendingNTLMTraffic", 2),
		)
		wdigest := items(d("UseLogonCredential", 0))
		pku2u := items(d("AllowOnlineID", 0))

		v := computeLsaNtlm(msv10, wdigest, pku2u)
		require.NotNil(t, v.AllowNullSessionFallback)
		assert.Equal(t, false, *v.AllowNullSessionFallback)
		require.NotNil(t, v.AuditReceivingNtlmTraffic)
		assert.Equal(t, int64(2), *v.AuditReceivingNtlmTraffic)
		require.NotNil(t, v.NtlmMinClientSec)
		assert.Equal(t, int64(537395200), *v.NtlmMinClientSec)
		require.NotNil(t, v.NtlmMinServerSec)
		assert.Equal(t, int64(537395200), *v.NtlmMinServerSec)
		require.NotNil(t, v.RestrictSendingNtlmTraffic)
		assert.Equal(t, int64(2), *v.RestrictSendingNtlmTraffic)
		// UseLogonCredential==false is the compliant value; absent must not look like false
		require.NotNil(t, v.UseLogonCredential)
		assert.Equal(t, false, *v.UseLogonCredential)
		require.NotNil(t, v.AllowOnlineId)
		assert.Equal(t, false, *v.AllowOnlineId)
	})

	t.Run("WDigest value absent stays null while MSV1_0 set", func(t *testing.T) {
		v := computeLsaNtlm(items(d("RestrictSendingNTLMTraffic", 2)), map[string]registry.RegistryKeyItem{}, map[string]registry.RegistryKeyItem{})
		require.NotNil(t, v.RestrictSendingNtlmTraffic)
		assert.Nil(t, v.UseLogonCredential)
		assert.Nil(t, v.AllowOnlineId)
	})
}

func TestComputeLsaSecureChannel(t *testing.T) {
	t.Run("empty keys yield all-null values", func(t *testing.T) {
		empty := map[string]registry.RegistryKeyItem{}
		v := computeLsaSecureChannel(empty, empty)
		assert.Nil(t, v.AuditNtlmInDomain)
		assert.Nil(t, v.BlockNetbiosDiscovery)
		assert.Nil(t, v.DisablePasswordChange)
		assert.Nil(t, v.MaximumPasswordAge)
		assert.Nil(t, v.RefusePasswordChange)
		assert.Nil(t, v.RequireSignOrSeal)
		assert.Nil(t, v.RequireStrongKey)
		assert.Nil(t, v.SealSecureChannel)
		assert.Nil(t, v.SignSecureChannel)
		assert.Nil(t, v.VulnerableChannelAllowList)
	})

	t.Run("BlockNetbiosDiscovery comes from the GPO policy key", func(t *testing.T) {
		netlogon := items(
			d("AuditNTLMInDomain", 7),
			d("DisablePasswordChange", 0),
			d("MaximumPasswordAge", 30),
			d("RefusePasswordChange", 0),
			d("RequireSignOrSeal", 1),
			d("RequireStrongKey", 1),
			d("SealSecureChannel", 1),
			d("SignSecureChannel", 1),
			s("vulnerablechannelallowlist", ""),
		)
		// BlockNetbiosDiscovery deliberately only set in the policy hive
		policy := items(d("BlockNetbiosDiscovery", 1))

		v := computeLsaSecureChannel(netlogon, policy)
		require.NotNil(t, v.AuditNtlmInDomain)
		assert.Equal(t, int64(7), *v.AuditNtlmInDomain)
		require.NotNil(t, v.BlockNetbiosDiscovery)
		assert.Equal(t, true, *v.BlockNetbiosDiscovery)
		require.NotNil(t, v.DisablePasswordChange)
		assert.Equal(t, false, *v.DisablePasswordChange)
		require.NotNil(t, v.MaximumPasswordAge)
		assert.Equal(t, int64(30), *v.MaximumPasswordAge)
		require.NotNil(t, v.RefusePasswordChange)
		assert.Equal(t, false, *v.RefusePasswordChange)
		require.NotNil(t, v.RequireSignOrSeal)
		assert.Equal(t, true, *v.RequireSignOrSeal)
		require.NotNil(t, v.RequireStrongKey)
		assert.Equal(t, true, *v.RequireStrongKey)
		require.NotNil(t, v.SealSecureChannel)
		assert.Equal(t, true, *v.SealSecureChannel)
		require.NotNil(t, v.SignSecureChannel)
		assert.Equal(t, true, *v.SignSecureChannel)
		require.NotNil(t, v.VulnerableChannelAllowList)
		assert.Equal(t, "", *v.VulnerableChannelAllowList)
	})

	t.Run("BlockNetbiosDiscovery absent from policy hive stays null", func(t *testing.T) {
		netlogon := items(d("RequireSignOrSeal", 1))
		v := computeLsaSecureChannel(netlogon, map[string]registry.RegistryKeyItem{})
		assert.Nil(t, v.BlockNetbiosDiscovery)
		require.NotNil(t, v.RequireSignOrSeal)
	})
}

func TestLsaIntFieldPtr(t *testing.T) {
	t.Run("nil pointer yields a null field", func(t *testing.T) {
		f := intFieldPtr(nil)
		assert.True(t, f.IsSet())
		assert.True(t, f.IsNull())
	})

	t.Run("non-nil pointer yields the value", func(t *testing.T) {
		val := int64(5)
		f := intFieldPtr(&val)
		assert.True(t, f.IsSet())
		assert.False(t, f.IsNull())
		assert.Equal(t, int64(5), f.Data)
	})

	t.Run("explicit zero is set and not null", func(t *testing.T) {
		val := int64(0)
		f := intFieldPtr(&val)
		assert.True(t, f.IsSet())
		assert.False(t, f.IsNull())
		assert.Equal(t, int64(0), f.Data)
	})
}

func TestLsaRegBoolPtr(t *testing.T) {
	m := items(d("RequireSignOrSeal", 1), d("DisablePasswordChange", 0))

	t.Run("non-zero DWORD returns true", func(t *testing.T) {
		p := regBoolPtr(m, "RequireSignOrSeal")
		require.NotNil(t, p)
		assert.True(t, *p)
	})

	t.Run("explicit 0 returns false, distinguishable from absent", func(t *testing.T) {
		p := regBoolPtr(m, "DisablePasswordChange")
		require.NotNil(t, p)
		assert.False(t, *p)
	})

	t.Run("absent value returns nil", func(t *testing.T) {
		assert.Nil(t, regBoolPtr(m, "DoesNotExist"))
	})
}

func TestLsaBoolFieldPtr(t *testing.T) {
	t.Run("nil pointer yields a null field", func(t *testing.T) {
		f := boolFieldPtr(nil)
		assert.True(t, f.IsSet())
		assert.True(t, f.IsNull())
	})

	t.Run("explicit false is set and not null", func(t *testing.T) {
		val := false
		f := boolFieldPtr(&val)
		assert.True(t, f.IsSet())
		assert.False(t, f.IsNull())
		assert.Equal(t, false, f.Data)
	})
}

func TestLsaStringFieldPtr(t *testing.T) {
	t.Run("nil pointer yields a null field", func(t *testing.T) {
		f := stringFieldPtr(nil)
		assert.True(t, f.IsSet())
		assert.True(t, f.IsNull())
	})

	t.Run("explicit empty string is set and not null", func(t *testing.T) {
		empty := ""
		f := stringFieldPtr(&empty)
		assert.True(t, f.IsSet())
		assert.False(t, f.IsNull())
		assert.Equal(t, "", f.Data)
	})
}

// isNullBool and isNullInt report whether the generated field representation of
// a nullable source pointer lands in the null state, which is how an absent
// registry value has to surface to MQL. The TValue is bound to a local first
// because IsNull has a pointer receiver.
func isNullBool(p *bool) bool {
	f := boolFieldPtr(p)
	return f.IsSet() && f.IsNull()
}

func isNullInt(p *int64) bool {
	f := intFieldPtr(p)
	return f.IsSet() && f.IsNull()
}

func TestComputeLsaFips(t *testing.T) {
	t.Run("absent value yields null, not a fabricated false", func(t *testing.T) {
		assert.Nil(t, computeLsaFips(map[string]registry.RegistryKeyItem{}))
		assert.Nil(t, computeLsaFips(items(d("SomethingElse", 1))))
		assert.True(t, isNullBool(computeLsaFips(map[string]registry.RegistryKeyItem{})))
	})

	t.Run("explicit 0 is false and distinguishable from absent", func(t *testing.T) {
		p := computeLsaFips(items(d("Enabled", 0)))
		require.NotNil(t, p)
		assert.False(t, *p)
		assert.False(t, isNullBool(p))
	})

	t.Run("enabled reads true", func(t *testing.T) {
		p := computeLsaFips(items(d("Enabled", 1)))
		require.NotNil(t, p)
		assert.True(t, *p)
	})
}

func TestComputeLsaKerberos(t *testing.T) {
	empty := map[string]registry.RegistryKeyItem{}

	t.Run("absent in both locations yields null everywhere", func(t *testing.T) {
		v := computeLsaKerberos(empty, empty)
		assert.Nil(t, v.SupportedEncryptionTypes)
		assert.Nil(t, v.AllowsDesCbcCrc)
		assert.Nil(t, v.AllowsDesCbcMd5)
		assert.Nil(t, v.AllowsRc4Hmac)
		assert.Nil(t, v.AllowsAes128)
		assert.Nil(t, v.AllowsAes256)

		// an absent mask must not decode to "every type denied": that would
		// report DES and RC4 as blocked on a host that never configured the
		// policy and is in fact using the Windows default set
		assert.True(t, isNullInt(v.SupportedEncryptionTypes))
		assert.True(t, isNullBool(v.AllowsRc4Hmac))
		assert.True(t, isNullBool(v.AllowsAes256))
	})

	t.Run("the policy location wins over the legacy location", func(t *testing.T) {
		v := computeLsaKerberos(
			items(d("SupportedEncryptionTypes", 0x18)),
			items(d("SupportedEncryptionTypes", 0x4)),
		)
		require.NotNil(t, v.SupportedEncryptionTypes)
		assert.Equal(t, int64(0x18), *v.SupportedEncryptionTypes)
		require.NotNil(t, v.AllowsRc4Hmac)
		assert.False(t, *v.AllowsRc4Hmac)
	})

	t.Run("the legacy location is used when the policy value is absent", func(t *testing.T) {
		v := computeLsaKerberos(empty, items(d("SupportedEncryptionTypes", 0x4)))
		require.NotNil(t, v.SupportedEncryptionTypes)
		assert.Equal(t, int64(0x4), *v.SupportedEncryptionTypes)
		require.NotNil(t, v.AllowsRc4Hmac)
		assert.True(t, *v.AllowsRc4Hmac)
	})

	tests := []struct {
		name                                         string
		mask                                         int64
		desCbcCrc, desCbcMd5, rc4Hmac, aes128, ae256 bool
	}{
		{"legacy only (0x7)", 0x7, true, true, true, false, false},
		{"AES only (0x18)", 0x18, false, false, false, true, true},
		{"transitional RC4 plus AES (0x1C)", 0x1C, false, false, true, true, true},
		{"explicit zero denies every known type", 0x0, false, false, false, false, false},
		{"single legacy bit (0x1)", 0x1, true, false, false, false, false},
		{"AES256 alone (0x10)", 0x10, false, false, false, false, true},
		// 0x7FFFFFF8 is the hardened value the Windows baseline expects: every
		// reserved future-type bit is set alongside AES, and the three legacy
		// bits are clear. The unknown high bits must not bleed into any answer.
		{"future-type bits set with AES (0x7FFFFFF8)", 0x7FFFFFF8, false, false, false, true, true},
		// the same shape with a legacy bit deliberately left on, so a bug that
		// widened the mask would show up as DES reading false
		{"future-type bits set with DES still on (0x7FFFFFF9)", 0x7FFFFFF9, true, false, false, true, true},
		// 0x80000000 is documented as "use Windows default values"; it is an
		// unknown bit as far as the decode is concerned and must be ignored
		{"top bit set alongside AES (0x80000018)", 0x80000018, false, false, false, true, true},
		{"top bit alone (0x80000000)", 0x80000000, false, false, false, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := computeLsaKerberos(items(d("SupportedEncryptionTypes", tc.mask)), empty)

			require.NotNil(t, v.SupportedEncryptionTypes)
			assert.Equal(t, tc.mask, *v.SupportedEncryptionTypes, "the raw mask is reported unchanged")

			for _, f := range []struct {
				name string
				got  *bool
				want bool
			}{
				{"desCbcCrc", v.AllowsDesCbcCrc, tc.desCbcCrc},
				{"desCbcMd5", v.AllowsDesCbcMd5, tc.desCbcMd5},
				{"rc4Hmac", v.AllowsRc4Hmac, tc.rc4Hmac},
				{"aes128", v.AllowsAes128, tc.aes128},
				{"aes256", v.AllowsAes256, tc.ae256},
			} {
				require.NotNil(t, f.got, f.name)
				assert.Equal(t, f.want, *f.got, f.name)
			}
		})
	}
}

func TestComputeLsaLdap(t *testing.T) {
	empty := map[string]registry.RegistryKeyItem{}

	t.Run("no NTDS key leaves the server settings null, not zero", func(t *testing.T) {
		// the shape of a member server or standalone host: the NTDS service is
		// not installed, so the key is missing entirely. Reporting 0 here would
		// claim "LDAP signing not required" about a setting that does not apply.
		v := computeLsaLdap(empty, empty)
		assert.Nil(t, v.ServerIntegrity)
		assert.Nil(t, v.EnforceChannelBinding)
		assert.Nil(t, v.ClientIntegrity)

		assert.True(t, isNullInt(v.ServerIntegrity))
		assert.True(t, isNullInt(v.EnforceChannelBinding))
		assert.True(t, isNullInt(v.ClientIntegrity))
	})

	t.Run("client setting resolves while the server settings stay null", func(t *testing.T) {
		v := computeLsaLdap(empty, items(d("LDAPClientIntegrity", 1)))
		require.NotNil(t, v.ClientIntegrity)
		assert.Equal(t, int64(1), *v.ClientIntegrity)
		assert.Nil(t, v.ServerIntegrity)
		assert.Nil(t, v.EnforceChannelBinding)
	})

	t.Run("a domain controller resolves every value", func(t *testing.T) {
		v := computeLsaLdap(
			items(d("LDAPServerIntegrity", 2), d("LdapEnforceChannelBinding", 2)),
			items(d("LDAPClientIntegrity", 2)),
		)
		require.NotNil(t, v.ServerIntegrity)
		assert.Equal(t, int64(2), *v.ServerIntegrity)
		require.NotNil(t, v.EnforceChannelBinding)
		assert.Equal(t, int64(2), *v.EnforceChannelBinding)
		require.NotNil(t, v.ClientIntegrity)
		assert.Equal(t, int64(2), *v.ClientIntegrity)
	})

	t.Run("explicit 0 is reported as 0, not as null", func(t *testing.T) {
		v := computeLsaLdap(
			items(d("LDAPServerIntegrity", 0), d("LdapEnforceChannelBinding", 0)),
			items(d("LDAPClientIntegrity", 0)),
		)
		require.NotNil(t, v.ServerIntegrity)
		assert.Equal(t, int64(0), *v.ServerIntegrity)
		assert.False(t, isNullInt(v.ServerIntegrity))
		require.NotNil(t, v.EnforceChannelBinding)
		assert.False(t, isNullInt(v.EnforceChannelBinding))
		require.NotNil(t, v.ClientIntegrity)
		assert.False(t, isNullInt(v.ClientIntegrity))
	})

	t.Run("the server settings are read from NTDS, not from the client key", func(t *testing.T) {
		// a value planted on the wrong key must not satisfy the field
		v := computeLsaLdap(empty, items(d("LDAPServerIntegrity", 2)))
		assert.Nil(t, v.ServerIntegrity)
	})
}
