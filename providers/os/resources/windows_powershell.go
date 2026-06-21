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

// Registry locations that back the PowerShell logging policy. These are GPO-only
// values under the PowerShell policy key; there is no effective fallback, so
// when a value is absent the corresponding field is null (distinguishable from
// an explicit 0).
const (
	powershellPolicyPath        = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\PowerShell`
	powershellScriptBlockPath   = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\PowerShell\ScriptBlockLogging`
	powershellTranscriptionPath = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\PowerShell\Transcription`
)

func (r *mqlWindowsPowershell) id() (string, error) {
	return "windows.powershell", nil
}

func (r *mqlWindowsPowershellScriptBlockLogging) id() (string, error) {
	return "windows.powershell.scriptBlockLogging", nil
}

func (r *mqlWindowsPowershellTranscription) id() (string, error) {
	return "windows.powershell.transcription", nil
}

// readPowershellKey reads a single registry key and returns its values keyed by
// the lower-cased value name. A missing key yields an empty map rather than an
// error, so absent values resolve to null.
func (r *mqlWindowsPowershell) readPowershellKey(path string) (map[string]registry.RegistryKeyItem, error) {
	o, err := CreateResource(r.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (e.g. no Group Policy configured); treat it
		// as empty so absent values resolve to null
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

// powershellBoolPtr returns a pointer to the boolean interpretation of a
// registry DWORD (true for any non-zero value), or nil when the value name is
// absent. Returning a pointer keeps "not configured" (nil) distinguishable from
// an explicit false. Pure function for unit testing.
func powershellBoolPtr(items map[string]registry.RegistryKeyItem, name string) *bool {
	if it, ok := items[strings.ToLower(name)]; ok {
		v := it.Value.Number != 0
		return &v
	}
	return nil
}

// powershellStringPtr returns a pointer to the string value of a registry value,
// or nil when the value name is absent. Returning a pointer keeps "not
// configured" (nil) distinguishable from an explicit empty string. Pure function
// for unit testing.
func powershellStringPtr(items map[string]registry.RegistryKeyItem, name string) *string {
	if it, ok := items[strings.ToLower(name)]; ok {
		v := it.Value.String
		return &v
	}
	return nil
}

func (r *mqlWindowsPowershell) scriptBlockLogging() (*mqlWindowsPowershellScriptBlockLogging, error) {
	items, err := r.readPowershellKey(powershellScriptBlockPath)
	if err != nil {
		return nil, err
	}

	o, err := CreateResource(r.MqlRuntime, "windows.powershell.scriptBlockLogging", map[string]*llx.RawData{
		"__id":    llx.StringData("windows.powershell.scriptBlockLogging"),
		"enabled": llx.BoolDataPtr(powershellBoolPtr(items, "EnableScriptBlockLogging")),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsPowershellScriptBlockLogging), nil
}

func (r *mqlWindowsPowershell) transcription() (*mqlWindowsPowershellTranscription, error) {
	items, err := r.readPowershellKey(powershellTranscriptionPath)
	if err != nil {
		return nil, err
	}

	o, err := CreateResource(r.MqlRuntime, "windows.powershell.transcription", map[string]*llx.RawData{
		"__id":    llx.StringData("windows.powershell.transcription"),
		"enabled": llx.BoolDataPtr(powershellBoolPtr(items, "EnableTranscripting")),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsPowershellTranscription), nil
}

func (r *mqlWindowsPowershell) executionPolicy() (string, error) {
	items, err := r.readPowershellKey(powershellPolicyPath)
	if err != nil {
		return "", err
	}
	v := powershellStringPtr(items, "ExecutionPolicy")
	if v == nil {
		// the value is absent: set the field null so callers can distinguish
		// "not configured" from an explicit empty string. GetOrCompute respects
		// a field that the resolver sets proactively.
		r.ExecutionPolicy.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return *v, nil
}
