// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/resources/windows"
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

// mqlWindowsWinrmServiceInternal caches the one WS-Management read behind both
// address filters, so a query that reads the IPv4 and the IPv6 filter costs a
// single remote command rather than two.
type mqlWindowsWinrmServiceInternal struct {
	lock    sync.Mutex
	fetched bool
	config  *windows.WinRMConfig
}

// readWinRMConfig reads the live WS-Management client and service settings.
//
// These have no registry equivalent that can be trusted. The WinRM service
// applies the Group Policy key to its own configuration, so this value already
// carries a policy setting, while reading the policy key alone would miss a
// TrustedHosts set locally with `winrm set winrm/config/client`, which is the
// configuration actually worth finding.
func readWinRMConfig(runtime *plugin.Runtime) (*windows.WinRMConfig, error) {
	stdout, err := runWindowsPowerShell(runtime, windows.PSGetWinRMConfig, "read the WinRM configuration")
	if err != nil {
		return nil, err
	}
	return windows.ParseWinRMConfig(stdout)
}

func (r *mqlWindowsWinrmClient) trustedHosts() (string, error) {
	config, err := readWinRMConfig(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return string(config.Client.TrustedHosts), nil
}

// serviceConfig reads the WS-Management service settings once and caches them.
//
// The guard is read under the lock, never before it, so a racing accessor
// cannot see the flag set and the pointer still nil. The lock is uncontended
// in the common case and the work it guards is a remote command.
func (r *mqlWindowsWinrmService) serviceConfig() (*windows.WinRMConfig, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched {
		return r.config, nil
	}

	config, err := readWinRMConfig(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	r.config = config
	r.fetched = true
	return r.config, nil
}

func (r *mqlWindowsWinrmService) ipv4Filter() (string, error) {
	config, err := r.serviceConfig()
	if err != nil {
		return "", err
	}
	return string(config.Service.IPv4Filter), nil
}

func (r *mqlWindowsWinrmService) ipv6Filter() (string, error) {
	config, err := r.serviceConfig()
	if err != nil {
		return "", err
	}
	return string(config.Service.IPv6Filter), nil
}

// listeners enumerates the configured WS-Management listeners.
//
// A host with no listener reports an empty list; a configuration that cannot
// be read is an error. Collapsing the two would let "no listener accepts
// unencrypted HTTP" pass on a host whose listener configuration nobody managed
// to read.
func (r *mqlWindowsWinrm) listeners() ([]any, error) {
	stdout, err := runWindowsPowerShell(r.MqlRuntime, windows.PSGetWinRMListeners, "enumerate the WinRM listeners")
	if err != nil {
		return nil, err
	}

	list, err := windows.ParseWinRMListeners(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(list))
	for _, l := range list {
		o, err := CreateResource(r.MqlRuntime, "windows.winrm.listener", map[string]*llx.RawData{
			// one host carries an HTTP and an HTTPS listener on the same
			// address, and two listeners on different addresses with the same
			// transport, so both dimensions are in the id
			"__id":                  llx.StringData(l.ID()),
			"transport":             llx.StringData(string(l.Transport)),
			"port":                  llx.IntData(l.PortNumber()),
			"address":               llx.StringData(string(l.Address)),
			"enabled":               llx.BoolData(l.IsEnabled()),
			"certificateThumbprint": llx.StringData(string(l.CertificateThumbprint)),
			"hostname":              llx.StringData(string(l.Hostname)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

// windows.winrm.client and windows.winrm.service are both a field path on
// windows.winrm and a resource name in their own right. The compiler resolves
// the longest matching resource name before it considers a field, so the dotted
// form instantiates the sub-resource directly and the parent's accessor never
// runs. The fields the parent populates are then never set, and the query
// reports null for each one with "provider returned no data and no error".
//
// A null is worse than an error here, because a check reading "the service does
// not allow unencrypted traffic" off a null does not fail on a service that
// allows it.
//
// Delegating to the parent's accessor fills the resource in. The block form
// `windows.winrm { service { ... } }` binds the field instead of resolving a
// resource name and was never affected. When the resource is created normally
// by the parent it carries an __id and this is a no-op.
func initWindowsWinrmChild[T plugin.Resource](
	runtime *plugin.Runtime,
	args map[string]*llx.RawData,
	get func(*mqlWindowsWinrm) *plugin.TValue[T],
) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["__id"]; ok {
		return args, nil, nil
	}
	parent, err := CreateResource(runtime, "windows.winrm", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	v := get(parent.(*mqlWindowsWinrm))
	if v.Error != nil {
		return nil, nil, v.Error
	}
	return args, v.Data, nil
}

func initWindowsWinrmClient(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initWindowsWinrmChild(runtime, args, (*mqlWindowsWinrm).GetClient)
}

func initWindowsWinrmService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initWindowsWinrmChild(runtime, args, (*mqlWindowsWinrm).GetService)
}
