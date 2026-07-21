// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry locations that back the WinRM (Windows Remote Management) policy.
// The client, service, and WinRS settings are GPO-only values under the WinRM
// policy key; there is no per-listener effective fallback, so when a value is
// absent the documented Windows default applies. The service start mode comes
// from the WinRM service definition.
const (
	winrmClientPath       = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\WinRM\Client`
	winrmServicePath      = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service`
	winrmServiceWinRSPath = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service\WinRS`
	winrmServiceStartPath = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\WinRM`
)

// Documented Windows defaults applied when a policy value is absent. These are
// GPO-only values with no per-listener effective fallback.
const (
	winrmServiceStartDefault = 3 // manual
)

func (r *mqlWindowsWinrm) id() (string, error) {
	return "windows.winrm", nil
}

func (r *mqlWindowsWinrmClient) id() (string, error) {
	return "windows.winrm.client", nil
}

func (r *mqlWindowsWinrmService) id() (string, error) {
	return "windows.winrm.service", nil
}

// readWinRMKey reads a single registry key and returns its numeric values keyed
// by the lower-cased value name. A missing key yields an empty map rather than
// an error, so resolution falls through to the documented default.
func (r *mqlWindowsWinrm) readWinRMKey(path string) (map[string]int64, error) {
	o, err := CreateResource(r.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (e.g. no Group Policy configured); treat it
		// as empty so resolution falls through to the documented default
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return map[string]int64{}, nil
		}
		return nil, err
	}

	res := make(map[string]int64, len(entries))
	for i := range entries {
		res[strings.ToLower(entries[i].Key)] = entries[i].Value.Number
	}
	return res, nil
}

// winrmBool resolves a WinRM DWORD that toggles on the value 1, applying the
// documented default when the value name is absent. It is split out as a pure
// function so the precedence can be unit tested without a live registry.
func winrmBool(items map[string]int64, name string, def bool) bool {
	if v, ok := items[strings.ToLower(name)]; ok {
		return v == 1
	}
	return def
}

// computeWinRMClient derives the WinRM client booleans from the raw registry
// values of the Client policy key. Pure function for unit testing.
func computeWinRMClient(items map[string]int64) (allowBasic, allowUnencryptedTraffic, allowDigest bool) {
	// Windows historically allows Basic auth, unencrypted traffic, and Digest
	// auth on the client when the policy is not configured.
	allowBasic = winrmBool(items, "AllowBasic", true)
	allowUnencryptedTraffic = winrmBool(items, "AllowUnencryptedTraffic", true)
	allowDigest = winrmBool(items, "AllowDigest", true)
	return
}

// computeWinRMService derives the WinRM service booleans from the raw registry
// values of the Service policy key and its WinRS subkey. Pure function for unit
// testing.
func computeWinRMService(service, winrs map[string]int64) (allowBasic, allowUnencryptedTraffic, disableRunAs, allowAutoConfig, allowRemoteShellAccess bool) {
	// Windows historically allows Basic auth and unencrypted traffic on the
	// service when the policy is not configured; RunAs storage is not disabled
	// and the listener is not auto-configured by default.
	allowBasic = winrmBool(service, "AllowBasic", true)
	allowUnencryptedTraffic = winrmBool(service, "AllowUnencryptedTraffic", true)
	disableRunAs = winrmBool(service, "DisableRunAs", false)
	allowAutoConfig = winrmBool(service, "AllowAutoConfig", false)
	// remote shell access is allowed by default
	allowRemoteShellAccess = winrmBool(winrs, "AllowRemoteShellAccess", true)
	return
}

// computeWinRMServiceStartMode returns the WinRM service Start DWORD, applying
// the documented default when the value is absent. Pure function for unit
// testing.
func computeWinRMServiceStartMode(items map[string]int64) int64 {
	if v, ok := items[strings.ToLower("Start")]; ok {
		return v
	}
	return winrmServiceStartDefault
}

func (r *mqlWindowsWinrm) client() (*mqlWindowsWinrmClient, error) {
	items, err := r.readWinRMKey(winrmClientPath)
	if err != nil {
		return nil, err
	}

	allowBasic, allowUnencryptedTraffic, allowDigest := computeWinRMClient(items)

	o, err := CreateResource(r.MqlRuntime, "windows.winrm.client", map[string]*llx.RawData{
		"__id":                    llx.StringData("windows.winrm.client"),
		"allowBasic":              llx.BoolData(allowBasic),
		"allowUnencryptedTraffic": llx.BoolData(allowUnencryptedTraffic),
		"allowDigest":             llx.BoolData(allowDigest),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsWinrmClient), nil
}

func (r *mqlWindowsWinrm) service() (*mqlWindowsWinrmService, error) {
	service, err := r.readWinRMKey(winrmServicePath)
	if err != nil {
		return nil, err
	}
	winrs, err := r.readWinRMKey(winrmServiceWinRSPath)
	if err != nil {
		return nil, err
	}

	allowBasic, allowUnencryptedTraffic, disableRunAs, allowAutoConfig, allowRemoteShellAccess := computeWinRMService(service, winrs)

	o, err := CreateResource(r.MqlRuntime, "windows.winrm.service", map[string]*llx.RawData{
		"__id":                    llx.StringData("windows.winrm.service"),
		"allowBasic":              llx.BoolData(allowBasic),
		"allowUnencryptedTraffic": llx.BoolData(allowUnencryptedTraffic),
		"disableRunAs":            llx.BoolData(disableRunAs),
		"allowAutoConfig":         llx.BoolData(allowAutoConfig),
		"allowRemoteShellAccess":  llx.BoolData(allowRemoteShellAccess),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsWinrmService), nil
}

func (r *mqlWindowsWinrm) serviceStartMode() (int64, error) {
	items, err := r.readWinRMKey(winrmServiceStartPath)
	if err != nil {
		return 0, err
	}
	return computeWinRMServiceStartMode(items), nil
}
