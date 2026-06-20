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

// lsaValues holds the extracted top-level Lsa values as nullable pointers.
type lsaValues struct {
	DisableDomainCreds          *int64
	EveryoneIncludesAnonymous   *int64
	ForceGuest                  *int64
	LimitBlankPasswordUse       *int64
	LmCompatibilityLevel        *int64
	NoLmHash                    *int64
	RestrictAnonymous           *int64
	RestrictAnonymousSam        *int64
	RestrictRemoteSam           *string
	RunAsPpl                    *int64
	SceNoApplyLegacyAuditPolicy *int64
	SubmitControl               *int64
	UseMachineId                *int64
}

// computeLsa extracts the top-level Lsa values from the raw registry items.
// Pure function for unit testing.
func computeLsa(items map[string]registry.RegistryKeyItem) lsaValues {
	return lsaValues{
		DisableDomainCreds:          regIntPtr(items, "DisableDomainCreds"),
		EveryoneIncludesAnonymous:   regIntPtr(items, "EveryoneIncludesAnonymous"),
		ForceGuest:                  regIntPtr(items, "ForceGuest"),
		LimitBlankPasswordUse:       regIntPtr(items, "LimitBlankPasswordUse"),
		LmCompatibilityLevel:        regIntPtr(items, "LmCompatibilityLevel"),
		NoLmHash:                    regIntPtr(items, "NoLMHash"),
		RestrictAnonymous:           regIntPtr(items, "RestrictAnonymous"),
		RestrictAnonymousSam:        regIntPtr(items, "RestrictAnonymousSAM"),
		RestrictRemoteSam:           regStringPtr(items, "restrictremotesam"),
		RunAsPpl:                    regIntPtr(items, "RunAsPPL"),
		SceNoApplyLegacyAuditPolicy: regIntPtr(items, "SCENoApplyLegacyAuditPolicy"),
		SubmitControl:               regIntPtr(items, "SubmitControl"),
		UseMachineId:                regIntPtr(items, "UseMachineId"),
	}
}

// lsaNtlmValues holds the extracted NTLM (MSV1_0 + WDigest + pku2u) values as
// nullable pointers.
type lsaNtlmValues struct {
	AllowNullSessionFallback   *int64
	AuditReceivingNtlmTraffic  *int64
	NtlmMinClientSec           *int64
	NtlmMinServerSec           *int64
	RestrictSendingNtlmTraffic *int64
	UseLogonCredential         *int64
	AllowOnlineId              *int64
}

// computeLsaNtlm extracts the NTLM settings from the MSV1_0, WDigest, and pku2u
// registry items. Pure function for unit testing.
func computeLsaNtlm(msv10, wdigest, pku2u map[string]registry.RegistryKeyItem) lsaNtlmValues {
	return lsaNtlmValues{
		AllowNullSessionFallback:   regIntPtr(msv10, "AllowNullSessionFallback"),
		AuditReceivingNtlmTraffic:  regIntPtr(msv10, "AuditReceivingNTLMTraffic"),
		NtlmMinClientSec:           regIntPtr(msv10, "NTLMMinClientSec"),
		NtlmMinServerSec:           regIntPtr(msv10, "NTLMMinServerSec"),
		RestrictSendingNtlmTraffic: regIntPtr(msv10, "RestrictSendingNTLMTraffic"),
		UseLogonCredential:         regIntPtr(wdigest, "UseLogonCredential"),
		AllowOnlineId:              regIntPtr(pku2u, "AllowOnlineID"),
	}
}

// lsaSecureChannelValues holds the extracted Netlogon secure-channel values as
// nullable pointers.
type lsaSecureChannelValues struct {
	AuditNtlmInDomain          *int64
	BlockNetbiosDiscovery      *int64
	DisablePasswordChange      *int64
	MaximumPasswordAge         *int64
	RefusePasswordChange       *int64
	RequireSignOrSeal          *int64
	RequireStrongKey           *int64
	SealSecureChannel          *int64
	SignSecureChannel          *int64
	VulnerableChannelAllowList *string
}

// computeLsaSecureChannel extracts the Netlogon secure-channel settings.
// BlockNetbiosDiscovery is sourced from the GPO policy hive while the remaining
// values come from the Services\Netlogon\Parameters key. Pure function for unit
// testing.
func computeLsaSecureChannel(netlogon, netlogonPolicy map[string]registry.RegistryKeyItem) lsaSecureChannelValues {
	return lsaSecureChannelValues{
		AuditNtlmInDomain:          regIntPtr(netlogon, "AuditNTLMInDomain"),
		BlockNetbiosDiscovery:      regIntPtr(netlogonPolicy, "BlockNetbiosDiscovery"),
		DisablePasswordChange:      regIntPtr(netlogon, "DisablePasswordChange"),
		MaximumPasswordAge:         regIntPtr(netlogon, "MaximumPasswordAge"),
		RefusePasswordChange:       regIntPtr(netlogon, "RefusePasswordChange"),
		RequireSignOrSeal:          regIntPtr(netlogon, "RequireSignOrSeal"),
		RequireStrongKey:           regIntPtr(netlogon, "RequireStrongKey"),
		SealSecureChannel:          regIntPtr(netlogon, "SealSecureChannel"),
		SignSecureChannel:          regIntPtr(netlogon, "SignSecureChannel"),
		VulnerableChannelAllowList: regStringPtr(netlogon, "vulnerablechannelallowlist"),
	}
}

func (r *mqlWindowsLsa) disableDomainCreds() (int64, error)          { return 0, r.populate() }
func (r *mqlWindowsLsa) everyoneIncludesAnonymous() (int64, error)   { return 0, r.populate() }
func (r *mqlWindowsLsa) forceGuest() (int64, error)                  { return 0, r.populate() }
func (r *mqlWindowsLsa) limitBlankPasswordUse() (int64, error)       { return 0, r.populate() }
func (r *mqlWindowsLsa) lmCompatibilityLevel() (int64, error)        { return 0, r.populate() }
func (r *mqlWindowsLsa) noLmHash() (int64, error)                    { return 0, r.populate() }
func (r *mqlWindowsLsa) restrictAnonymous() (int64, error)           { return 0, r.populate() }
func (r *mqlWindowsLsa) restrictAnonymousSam() (int64, error)        { return 0, r.populate() }
func (r *mqlWindowsLsa) restrictRemoteSam() (string, error)          { return "", r.populate() }
func (r *mqlWindowsLsa) runAsPpl() (int64, error)                    { return 0, r.populate() }
func (r *mqlWindowsLsa) sceNoApplyLegacyAuditPolicy() (int64, error) { return 0, r.populate() }
func (r *mqlWindowsLsa) submitControl() (int64, error)               { return 0, r.populate() }
func (r *mqlWindowsLsa) useMachineId() (int64, error)                { return 0, r.populate() }

// populate reads the Lsa key once and fills every top-level field. Each field
// accessor delegates here; the lazy-field machinery caches the results so the
// registry is read a single time regardless of how many fields are queried.
func (r *mqlWindowsLsa) populate() error {
	items, err := r.readLsaKey(lsaPath)
	if err != nil {
		return err
	}
	v := computeLsa(items)

	r.DisableDomainCreds = intFieldPtr(v.DisableDomainCreds)
	r.EveryoneIncludesAnonymous = intFieldPtr(v.EveryoneIncludesAnonymous)
	r.ForceGuest = intFieldPtr(v.ForceGuest)
	r.LimitBlankPasswordUse = intFieldPtr(v.LimitBlankPasswordUse)
	r.LmCompatibilityLevel = intFieldPtr(v.LmCompatibilityLevel)
	r.NoLmHash = intFieldPtr(v.NoLmHash)
	r.RestrictAnonymous = intFieldPtr(v.RestrictAnonymous)
	r.RestrictAnonymousSam = intFieldPtr(v.RestrictAnonymousSam)
	r.RestrictRemoteSam = stringFieldPtr(v.RestrictRemoteSam)
	r.RunAsPpl = intFieldPtr(v.RunAsPpl)
	r.SceNoApplyLegacyAuditPolicy = intFieldPtr(v.SceNoApplyLegacyAuditPolicy)
	r.SubmitControl = intFieldPtr(v.SubmitControl)
	r.UseMachineId = intFieldPtr(v.UseMachineId)
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
		"allowNullSessionFallback":   llx.IntDataPtr(v.AllowNullSessionFallback),
		"auditReceivingNtlmTraffic":  llx.IntDataPtr(v.AuditReceivingNtlmTraffic),
		"ntlmMinClientSec":           llx.IntDataPtr(v.NtlmMinClientSec),
		"ntlmMinServerSec":           llx.IntDataPtr(v.NtlmMinServerSec),
		"restrictSendingNtlmTraffic": llx.IntDataPtr(v.RestrictSendingNtlmTraffic),
		"useLogonCredential":         llx.IntDataPtr(v.UseLogonCredential),
		"allowOnlineId":              llx.IntDataPtr(v.AllowOnlineId),
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
		"blockNetbiosDiscovery":      llx.IntDataPtr(v.BlockNetbiosDiscovery),
		"disablePasswordChange":      llx.IntDataPtr(v.DisablePasswordChange),
		"maximumPasswordAge":         llx.IntDataPtr(v.MaximumPasswordAge),
		"refusePasswordChange":       llx.IntDataPtr(v.RefusePasswordChange),
		"requireSignOrSeal":          llx.IntDataPtr(v.RequireSignOrSeal),
		"requireStrongKey":           llx.IntDataPtr(v.RequireStrongKey),
		"sealSecureChannel":          llx.IntDataPtr(v.SealSecureChannel),
		"signSecureChannel":          llx.IntDataPtr(v.SignSecureChannel),
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

// stringFieldPtr converts a nullable *string into the generated
// plugin.TValue[string] field representation, marking the field null when the
// source pointer is nil.
func stringFieldPtr(v *string) plugin.TValue[string] {
	if v == nil {
		return plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
	return plugin.TValue[string]{Data: *v, State: plugin.StateIsSet}
}
