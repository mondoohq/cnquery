// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/registry"
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
		assert.Equal(t, int64(1), *v.DisableDomainCreds)
		require.NotNil(t, v.EveryoneIncludesAnonymous)
		assert.Equal(t, int64(0), *v.EveryoneIncludesAnonymous)
		require.NotNil(t, v.ForceGuest)
		assert.Equal(t, int64(0), *v.ForceGuest)
		require.NotNil(t, v.LimitBlankPasswordUse)
		assert.Equal(t, int64(1), *v.LimitBlankPasswordUse)
		require.NotNil(t, v.LmCompatibilityLevel)
		assert.Equal(t, int64(5), *v.LmCompatibilityLevel)
		require.NotNil(t, v.NoLmHash)
		assert.Equal(t, int64(1), *v.NoLmHash)
		require.NotNil(t, v.RestrictAnonymous)
		assert.Equal(t, int64(1), *v.RestrictAnonymous)
		require.NotNil(t, v.RestrictAnonymousSam)
		assert.Equal(t, int64(1), *v.RestrictAnonymousSam)
		require.NotNil(t, v.RestrictRemoteSam)
		assert.Equal(t, "O:BAG:BAD:(A;;RC;;;BA)", *v.RestrictRemoteSam)
		require.NotNil(t, v.RunAsPpl)
		assert.Equal(t, int64(1), *v.RunAsPpl)
		require.NotNil(t, v.SceNoApplyLegacyAuditPolicy)
		assert.Equal(t, int64(1), *v.SceNoApplyLegacyAuditPolicy)
		require.NotNil(t, v.SubmitControl)
		assert.Equal(t, int64(0), *v.SubmitControl)
		require.NotNil(t, v.UseMachineId)
		assert.Equal(t, int64(1), *v.UseMachineId)
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
		assert.Equal(t, int64(0), *v.AllowNullSessionFallback)
		require.NotNil(t, v.AuditReceivingNtlmTraffic)
		assert.Equal(t, int64(2), *v.AuditReceivingNtlmTraffic)
		require.NotNil(t, v.NtlmMinClientSec)
		assert.Equal(t, int64(537395200), *v.NtlmMinClientSec)
		require.NotNil(t, v.NtlmMinServerSec)
		assert.Equal(t, int64(537395200), *v.NtlmMinServerSec)
		require.NotNil(t, v.RestrictSendingNtlmTraffic)
		assert.Equal(t, int64(2), *v.RestrictSendingNtlmTraffic)
		// UseLogonCredential==0 is the compliant value; absent must not look like 0
		require.NotNil(t, v.UseLogonCredential)
		assert.Equal(t, int64(0), *v.UseLogonCredential)
		require.NotNil(t, v.AllowOnlineId)
		assert.Equal(t, int64(0), *v.AllowOnlineId)
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
		assert.Equal(t, int64(1), *v.BlockNetbiosDiscovery)
		require.NotNil(t, v.DisablePasswordChange)
		assert.Equal(t, int64(0), *v.DisablePasswordChange)
		require.NotNil(t, v.MaximumPasswordAge)
		assert.Equal(t, int64(30), *v.MaximumPasswordAge)
		require.NotNil(t, v.RefusePasswordChange)
		assert.Equal(t, int64(0), *v.RefusePasswordChange)
		require.NotNil(t, v.RequireSignOrSeal)
		assert.Equal(t, int64(1), *v.RequireSignOrSeal)
		require.NotNil(t, v.RequireStrongKey)
		assert.Equal(t, int64(1), *v.RequireStrongKey)
		require.NotNil(t, v.SealSecureChannel)
		assert.Equal(t, int64(1), *v.SealSecureChannel)
		require.NotNil(t, v.SignSecureChannel)
		assert.Equal(t, int64(1), *v.SignSecureChannel)
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
