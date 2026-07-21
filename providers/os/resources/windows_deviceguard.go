// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/registry"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry location that backs windows.deviceGuard. These are GPO-only DWORD
// values; every one is optional, so an absent value means "not configured"
// rather than 0.
const deviceGuardPolicyKey = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`

func (r *mqlWindowsDeviceGuard) id() (string, error) {
	return "windows.deviceGuard", nil
}

// readDeviceGuardKey reads the Device Guard policy key and returns its values as
// a name->item map (lower-cased keys). A missing key yields an empty map rather
// than an error, so every field falls through to null ("not configured").
func (w *mqlWindows) readDeviceGuardKey(path string) (map[string]registry.RegistryKeyItem, error) {
	o, err := CreateResource(w.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (e.g. no Group Policy configured); treat it
		// as empty so every value resolves to null
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

// regIntPtr returns a pointer to the numeric value of a registry item, or nil
// when the value name is absent. The pointer makes "not configured" (nil)
// distinguishable from an explicit 0.
func regIntPtr(items map[string]registry.RegistryKeyItem, name string) *int64 {
	if it, ok := items[strings.ToLower(name)]; ok {
		v := it.Value.Number
		return &v
	}
	return nil
}

// deviceGuardValues holds the extracted Device Guard DWORDs as nullable
// pointers. A nil pointer means the value name was absent from the registry
// ("not configured"), which the resource surfaces as a null field rather than
// 0. On/off settings are bools; graded settings are int64s.
type deviceGuardValues struct {
	virtualizationBasedSecurityEnabled *bool
	requirePlatformSecurityFeatures    *int64
	hypervisorEnforcedCodeIntegrity    *int64
	hvciMatRequired                    *bool
	credentialGuardConfig              *int64
	systemGuardLaunch                  *bool
	kernelShadowStacksLaunch           *int64
}

// computeDeviceGuard extracts the Device Guard policy values from the raw
// registry items. Each field is nil when its value name is absent, so callers
// can tell "not configured" from an explicit 0. Pure function for unit testing.
func computeDeviceGuard(items map[string]registry.RegistryKeyItem) deviceGuardValues {
	return deviceGuardValues{
		virtualizationBasedSecurityEnabled: regBoolPtr(items, "EnableVirtualizationBasedSecurity"),
		requirePlatformSecurityFeatures:    regIntPtr(items, "RequirePlatformSecurityFeatures"),
		hypervisorEnforcedCodeIntegrity:    regIntPtr(items, "HypervisorEnforcedCodeIntegrity"),
		hvciMatRequired:                    regBoolPtr(items, "HVCIMATRequired"),
		credentialGuardConfig:              regIntPtr(items, "LsaCfgFlags"),
		systemGuardLaunch:                  regBoolPtr(items, "ConfigureSystemGuardLaunch"),
		kernelShadowStacksLaunch:           regIntPtr(items, "ConfigureKernelShadowStacksLaunch"),
	}
}

func (w *mqlWindows) deviceGuard() (*mqlWindowsDeviceGuard, error) {
	items, err := w.readDeviceGuardKey(deviceGuardPolicyKey)
	if err != nil {
		return nil, err
	}

	v := computeDeviceGuard(items)

	o, err := CreateResource(w.MqlRuntime, "windows.deviceGuard", map[string]*llx.RawData{
		"__id":                               llx.StringData("windows.deviceGuard"),
		"virtualizationBasedSecurityEnabled": llx.BoolDataPtr(v.virtualizationBasedSecurityEnabled),
		"requirePlatformSecurityFeatures":    llx.IntDataPtr(v.requirePlatformSecurityFeatures),
		"hypervisorEnforcedCodeIntegrity":    llx.IntDataPtr(v.hypervisorEnforcedCodeIntegrity),
		"hvciMatRequired":                    llx.BoolDataPtr(v.hvciMatRequired),
		"credentialGuardConfig":              llx.IntDataPtr(v.credentialGuardConfig),
		"systemGuardLaunch":                  llx.BoolDataPtr(v.systemGuardLaunch),
		"kernelShadowStacksLaunch":           llx.IntDataPtr(v.kernelShadowStacksLaunch),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsDeviceGuard), nil
}
