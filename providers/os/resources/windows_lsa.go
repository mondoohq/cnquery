// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/registry"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry locations that back the Windows LSA (Local Security Authority)
// security policy. The Lsa key holds the bulk of the "Network access" /
// "Network security" security options; MSV1_0 and the WDigest provider hold the
// NTLM settings; and Netlogon\Parameters holds the secure-channel settings for
// domain members. BlockNetbiosDiscovery lives in the GPO policy hive rather
// than the Services hive, so it is read from its own key.
const (
	lsaPath               = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Lsa`
	lsaMSV10Path          = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0`
	lsaPKU2UPath          = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Lsa\pku2u`
	lsaWDigestPath        = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecurityProviders\WDigest`
	lsaNetlogonPath       = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`
	lsaNetlogonPolicyPath = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Netlogon\Parameters`
)

func (r *mqlWindowsLsa) id() (string, error) {
	return "windows.lsa", nil
}

func (r *mqlWindowsLsaNtlm) id() (string, error) {
	return "windows.lsa.ntlm", nil
}

func (r *mqlWindowsLsaSecureChannel) id() (string, error) {
	return "windows.lsa.secureChannel", nil
}

// readLsaKey reads a single registry key and returns its values as a name->item
// map (lower-cased keys). A missing key yields an empty map rather than an
// error, so an absent value resolves to null (distinguishable from an explicit
// 0). A genuine read failure is surfaced to the caller.
func (r *mqlWindowsLsa) readLsaKey(path string) (map[string]registry.RegistryKeyItem, error) {
	o, err := CreateResource(r.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (e.g. the value was never configured); treat
		// it as empty so the corresponding fields resolve to null
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return map[string]registry.RegistryKeyItem{}, nil
		}
		return nil, err
	}

	res := make(map[string]registry.RegistryKeyItem, len(entries))
	for i := range entries {
		res[strings.ToLower(entries[i].Key)] = entries[i]
	}
	return res, nil
}

// regStringPtr returns a pointer to the string value of a registry value, or
// nil when the value is absent. Used for REG_SZ values such as the
// restrictremotesam SDDL string and the vulnerable channel allow list.
func regStringPtr(items map[string]registry.RegistryKeyItem, name string) *string {
	if it, ok := items[strings.ToLower(name)]; ok {
		v := it.Value.String
		return &v
	}
	return nil
}

// regBoolPtr returns a pointer to the boolean interpretation of a registry
// DWORD (true for any non-zero value), or nil when the value is absent. Used
// for the on/off LSA settings whose registry value is a 0/1 DWORD.
func regBoolPtr(items map[string]registry.RegistryKeyItem, name string) *bool {
	if it, ok := items[strings.ToLower(name)]; ok {
		v := it.Value.Number != 0
		return &v
	}
	return nil
}

// lsaValues holds the extracted top-level Lsa values as nullable pointers.
// On/off settings are bools; graded settings are int64s.
type lsaValues struct {
	DisableDomainCreds          *bool
	EveryoneIncludesAnonymous   *bool
	ForceGuest                  *bool
	LimitBlankPasswordUse       *bool
	LmCompatibilityLevel        *int64
	NoLmHash                    *bool
	RestrictAnonymous           *int64
	RestrictAnonymousSam        *bool
	RestrictRemoteSam           *string
	RunAsPpl                    *int64
	SceNoApplyLegacyAuditPolicy *bool
	SubmitControl               *bool
	UseMachineId                *bool
}

// computeLsa extracts the top-level Lsa values from the raw registry items.
// Pure function for unit testing.
func computeLsa(items map[string]registry.RegistryKeyItem) lsaValues {
	return lsaValues{
		DisableDomainCreds:          regBoolPtr(items, "DisableDomainCreds"),
		EveryoneIncludesAnonymous:   regBoolPtr(items, "EveryoneIncludesAnonymous"),
		ForceGuest:                  regBoolPtr(items, "ForceGuest"),
		LimitBlankPasswordUse:       regBoolPtr(items, "LimitBlankPasswordUse"),
		LmCompatibilityLevel:        regIntPtr(items, "LmCompatibilityLevel"),
		NoLmHash:                    regBoolPtr(items, "NoLMHash"),
		RestrictAnonymous:           regIntPtr(items, "RestrictAnonymous"),
		RestrictAnonymousSam:        regBoolPtr(items, "RestrictAnonymousSAM"),
		RestrictRemoteSam:           regStringPtr(items, "restrictremotesam"),
		RunAsPpl:                    regIntPtr(items, "RunAsPPL"),
		SceNoApplyLegacyAuditPolicy: regBoolPtr(items, "SCENoApplyLegacyAuditPolicy"),
		SubmitControl:               regBoolPtr(items, "SubmitControl"),
		UseMachineId:                regBoolPtr(items, "UseMachineId"),
	}
}

// lsaNtlmValues holds the extracted NTLM (MSV1_0 + WDigest + pku2u) values as
// nullable pointers.
type lsaNtlmValues struct {
	AllowNullSessionFallback   *bool
	AuditReceivingNtlmTraffic  *int64
	NtlmMinClientSec           *int64
	NtlmMinServerSec           *int64
	RestrictSendingNtlmTraffic *int64
	UseLogonCredential         *bool
	AllowOnlineId              *bool
}

// computeLsaNtlm extracts the NTLM settings from the MSV1_0, WDigest, and pku2u
// registry items. Pure function for unit testing.
func computeLsaNtlm(msv10, wdigest, pku2u map[string]registry.RegistryKeyItem) lsaNtlmValues {
	return lsaNtlmValues{
		AllowNullSessionFallback:   regBoolPtr(msv10, "AllowNullSessionFallback"),
		AuditReceivingNtlmTraffic:  regIntPtr(msv10, "AuditReceivingNTLMTraffic"),
		NtlmMinClientSec:           regIntPtr(msv10, "NTLMMinClientSec"),
		NtlmMinServerSec:           regIntPtr(msv10, "NTLMMinServerSec"),
		RestrictSendingNtlmTraffic: regIntPtr(msv10, "RestrictSendingNTLMTraffic"),
		UseLogonCredential:         regBoolPtr(wdigest, "UseLogonCredential"),
		AllowOnlineId:              regBoolPtr(pku2u, "AllowOnlineID"),
	}
}

// lsaSecureChannelValues holds the extracted Netlogon secure-channel values as
// nullable pointers.
type lsaSecureChannelValues struct {
	AuditNtlmInDomain          *int64
	BlockNetbiosDiscovery      *bool
	DisablePasswordChange      *bool
	MaximumPasswordAge         *int64
	RefusePasswordChange       *bool
	RequireSignOrSeal          *bool
	RequireStrongKey           *bool
	SealSecureChannel          *bool
	SignSecureChannel          *bool
	VulnerableChannelAllowList *string
}

// computeLsaSecureChannel extracts the Netlogon secure-channel settings.
// BlockNetbiosDiscovery is sourced from the GPO policy hive while the remaining
// values come from the Services\Netlogon\Parameters key. Pure function for unit
// testing.
func computeLsaSecureChannel(netlogon, netlogonPolicy map[string]registry.RegistryKeyItem) lsaSecureChannelValues {
	return lsaSecureChannelValues{
		AuditNtlmInDomain:          regIntPtr(netlogon, "AuditNTLMInDomain"),
		BlockNetbiosDiscovery:      regBoolPtr(netlogonPolicy, "BlockNetbiosDiscovery"),
		DisablePasswordChange:      regBoolPtr(netlogon, "DisablePasswordChange"),
		MaximumPasswordAge:         regIntPtr(netlogon, "MaximumPasswordAge"),
		RefusePasswordChange:       regBoolPtr(netlogon, "RefusePasswordChange"),
		RequireSignOrSeal:          regBoolPtr(netlogon, "RequireSignOrSeal"),
		RequireStrongKey:           regBoolPtr(netlogon, "RequireStrongKey"),
		SealSecureChannel:          regBoolPtr(netlogon, "SealSecureChannel"),
		SignSecureChannel:          regBoolPtr(netlogon, "SignSecureChannel"),
		VulnerableChannelAllowList: regStringPtr(netlogon, "vulnerablechannelallowlist"),
	}
}

func (r *mqlWindowsLsa) disableDomainCreds() (bool, error)          { return false, r.populate() }
func (r *mqlWindowsLsa) everyoneIncludesAnonymous() (bool, error)   { return false, r.populate() }
func (r *mqlWindowsLsa) forceGuest() (bool, error)                  { return false, r.populate() }
func (r *mqlWindowsLsa) limitBlankPasswordUse() (bool, error)       { return false, r.populate() }
func (r *mqlWindowsLsa) lmCompatibilityLevel() (int64, error)       { return 0, r.populate() }
func (r *mqlWindowsLsa) noLmHash() (bool, error)                    { return false, r.populate() }
func (r *mqlWindowsLsa) restrictAnonymous() (int64, error)          { return 0, r.populate() }
func (r *mqlWindowsLsa) restrictAnonymousSam() (bool, error)        { return false, r.populate() }
func (r *mqlWindowsLsa) restrictRemoteSam() (string, error)         { return "", r.populate() }
func (r *mqlWindowsLsa) runAsPpl() (int64, error)                   { return 0, r.populate() }
func (r *mqlWindowsLsa) sceNoApplyLegacyAuditPolicy() (bool, error) { return false, r.populate() }
func (r *mqlWindowsLsa) submitControl() (bool, error)               { return false, r.populate() }
func (r *mqlWindowsLsa) useMachineId() (bool, error)                { return false, r.populate() }

// populate reads the Lsa key once and fills every top-level field. Each field
// accessor delegates here; the lazy-field machinery caches the results so the
// registry is read a single time regardless of how many fields are queried.
func (r *mqlWindowsLsa) populate() error {
	items, err := r.readLsaKey(lsaPath)
	if err != nil {
		return err
	}
	v := computeLsa(items)

	r.DisableDomainCreds = boolFieldPtr(v.DisableDomainCreds)
	r.EveryoneIncludesAnonymous = boolFieldPtr(v.EveryoneIncludesAnonymous)
	r.ForceGuest = boolFieldPtr(v.ForceGuest)
	r.LimitBlankPasswordUse = boolFieldPtr(v.LimitBlankPasswordUse)
	r.LmCompatibilityLevel = intFieldPtr(v.LmCompatibilityLevel)
	r.NoLmHash = boolFieldPtr(v.NoLmHash)
	r.RestrictAnonymous = intFieldPtr(v.RestrictAnonymous)
	r.RestrictAnonymousSam = boolFieldPtr(v.RestrictAnonymousSam)
	r.RestrictRemoteSam = stringFieldPtr(v.RestrictRemoteSam)
	r.RunAsPpl = intFieldPtr(v.RunAsPpl)
	r.SceNoApplyLegacyAuditPolicy = boolFieldPtr(v.SceNoApplyLegacyAuditPolicy)
	r.SubmitControl = boolFieldPtr(v.SubmitControl)
	r.UseMachineId = boolFieldPtr(v.UseMachineId)
	return nil
}

func (r *mqlWindowsLsa) ntlm() (*mqlWindowsLsaNtlm, error) {
	msv10, err := r.readLsaKey(lsaMSV10Path)
	if err != nil {
		return nil, err
	}
	wdigest, err := r.readLsaKey(lsaWDigestPath)
	if err != nil {
		return nil, err
	}
	pku2u, err := r.readLsaKey(lsaPKU2UPath)
	if err != nil {
		return nil, err
	}

	v := computeLsaNtlm(msv10, wdigest, pku2u)

	o, err := CreateResource(r.MqlRuntime, "windows.lsa.ntlm", map[string]*llx.RawData{
		"__id":                       llx.StringData("windows.lsa.ntlm"),
		"allowNullSessionFallback":   llx.BoolDataPtr(v.AllowNullSessionFallback),
		"auditReceivingNtlmTraffic":  llx.IntDataPtr(v.AuditReceivingNtlmTraffic),
		"ntlmMinClientSec":           llx.IntDataPtr(v.NtlmMinClientSec),
		"ntlmMinServerSec":           llx.IntDataPtr(v.NtlmMinServerSec),
		"restrictSendingNtlmTraffic": llx.IntDataPtr(v.RestrictSendingNtlmTraffic),
		"useLogonCredential":         llx.BoolDataPtr(v.UseLogonCredential),
		"allowOnlineId":              llx.BoolDataPtr(v.AllowOnlineId),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsLsaNtlm), nil
}

func (r *mqlWindowsLsa) secureChannel() (*mqlWindowsLsaSecureChannel, error) {
	netlogon, err := r.readLsaKey(lsaNetlogonPath)
	if err != nil {
		return nil, err
	}
	netlogonPolicy, err := r.readLsaKey(lsaNetlogonPolicyPath)
	if err != nil {
		return nil, err
	}

	v := computeLsaSecureChannel(netlogon, netlogonPolicy)

	o, err := CreateResource(r.MqlRuntime, "windows.lsa.secureChannel", map[string]*llx.RawData{
		"__id":                       llx.StringData("windows.lsa.secureChannel"),
		"auditNtlmInDomain":          llx.IntDataPtr(v.AuditNtlmInDomain),
		"blockNetbiosDiscovery":      llx.BoolDataPtr(v.BlockNetbiosDiscovery),
		"disablePasswordChange":      llx.BoolDataPtr(v.DisablePasswordChange),
		"maximumPasswordAge":         llx.IntDataPtr(v.MaximumPasswordAge),
		"refusePasswordChange":       llx.BoolDataPtr(v.RefusePasswordChange),
		"requireSignOrSeal":          llx.BoolDataPtr(v.RequireSignOrSeal),
		"requireStrongKey":           llx.BoolDataPtr(v.RequireStrongKey),
		"sealSecureChannel":          llx.BoolDataPtr(v.SealSecureChannel),
		"signSecureChannel":          llx.BoolDataPtr(v.SignSecureChannel),
		"vulnerableChannelAllowList": llx.StringDataPtr(v.VulnerableChannelAllowList),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsLsaSecureChannel), nil
}

// intFieldPtr converts a nullable *int64 into the generated plugin.TValue[int64]
// field representation, marking the field null when the source pointer is nil.
func intFieldPtr(v *int64) plugin.TValue[int64] {
	if v == nil {
		return plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
	return plugin.TValue[int64]{Data: *v, State: plugin.StateIsSet}
}

// boolFieldPtr converts a nullable *bool into the generated plugin.TValue[bool]
// field representation, marking the field null when the source pointer is nil.
func boolFieldPtr(v *bool) plugin.TValue[bool] {
	if v == nil {
		return plugin.TValue[bool]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
	return plugin.TValue[bool]{Data: *v, State: plugin.StateIsSet}
}

// stringFieldPtr converts a nullable *string into the generated
// plugin.TValue[string] field representation, marking the field null when the
// source pointer is nil.
func stringFieldPtr(v *string) plugin.TValue[string] {
	if v == nil {
		return plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
	return plugin.TValue[string]{Data: *v, State: plugin.StateIsSet}
}
