// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/registry"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry locations backing windows.spooler. The Group Policy printer keys
// hold the hardening-relevant settings; the Control\Print key and the spooler
// service key hold the machine-wide RPC privacy setting and the service start
// type respectively.
const (
	spoolerPrintersPath     = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows NT\Printers`
	spoolerPointAndPrintKey = spoolerPrintersPath + `\PointAndPrint`
	spoolerRpcKey           = spoolerPrintersPath + `\RPC`
	spoolerIppKey           = spoolerPrintersPath + `\IPP`
	spoolerWppKey           = spoolerPrintersPath + `\WPP`
	spoolerControlPrintKey  = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Print`
	spoolerLanManServersKey = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Print\Providers\LanMan Print Services\Servers`
	spoolerServiceKey       = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Spooler`
)

func (r *mqlWindowsSpooler) id() (string, error) {
	return "windows.spooler", nil
}

func (r *mqlWindowsSpoolerPointAndPrint) id() (string, error) {
	return "windows.spooler.pointAndPrint", nil
}

func (r *mqlWindowsSpoolerRpc) id() (string, error) {
	return "windows.spooler.rpc", nil
}

func (r *mqlWindowsSpoolerIpp) id() (string, error) {
	return "windows.spooler.ipp", nil
}

// spoolerRegistry caches the values of every registry key that backs the
// spooler resource so all fields and sub-resources share a single set of
// registry reads.
type spoolerRegistry struct {
	printers      map[string]registry.RegistryKeyItem
	pointAndPrint map[string]registry.RegistryKeyItem
	rpc           map[string]registry.RegistryKeyItem
	ipp           map[string]registry.RegistryKeyItem
	wpp           map[string]registry.RegistryKeyItem
	controlPrint  map[string]registry.RegistryKeyItem
	lanManServers map[string]registry.RegistryKeyItem
	service       map[string]registry.RegistryKeyItem
}

type mqlWindowsSpoolerInternal struct {
	lock    sync.Mutex
	loaded  bool
	reg     *spoolerRegistry
	loadErr error
}

// readSpoolerKey reads a single registry key, returning its values keyed by the
// lower-cased value name. A missing key yields an empty map rather than an
// error so callers treat absent settings as "not configured".
func (r *mqlWindowsSpooler) readSpoolerKey(path string) (map[string]registry.RegistryKeyItem, error) {
	o, err := CreateResource(r.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
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

// load reads every backing registry key exactly once and caches the result.
func (r *mqlWindowsSpooler) load() (*spoolerRegistry, error) {
	if r.loaded {
		return r.reg, r.loadErr
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.loaded {
		return r.reg, r.loadErr
	}

	reg := &spoolerRegistry{}
	reads := []struct {
		path string
		dst  *map[string]registry.RegistryKeyItem
	}{
		{spoolerPrintersPath, &reg.printers},
		{spoolerPointAndPrintKey, &reg.pointAndPrint},
		{spoolerRpcKey, &reg.rpc},
		{spoolerIppKey, &reg.ipp},
		{spoolerWppKey, &reg.wpp},
		{spoolerControlPrintKey, &reg.controlPrint},
		{spoolerLanManServersKey, &reg.lanManServers},
		{spoolerServiceKey, &reg.service},
	}
	for _, rd := range reads {
		items, err := r.readSpoolerKey(rd.path)
		if err != nil {
			r.loaded = true
			r.loadErr = err
			return nil, err
		}
		*rd.dst = items
	}

	r.reg = reg
	r.loaded = true
	return reg, nil
}

// regBoolDefault resolves a DWORD that toggles on the value 1 into a bool,
// falling back to def when the value is absent.
func regBoolDefault(items map[string]registry.RegistryKeyItem, name string, def bool) bool {
	if it, ok := items[strings.ToLower(name)]; ok {
		return it.Value.Number == 1
	}
	return def
}

// resolveNullableInt backs a lazily-computed int field that must be able to
// render as MQL null. When v is nil the field is set null proactively (so
// GetOrCompute returns the null TValue); otherwise the value is returned for
// GetOrCompute to cache. This keeps "not configured" distinct from an explicit
// 0 for fields whose compliant value is 0.
func resolveNullableInt(field *plugin.TValue[int64], v *int64) (int64, error) {
	if v == nil {
		*field = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
		return 0, nil
	}
	return *v, nil
}

// --- top-level fields -------------------------------------------------------

func (r *mqlWindowsSpooler) startMode() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.StartMode, spoolerStartMode(reg.service))
}

// spoolerStartMode extracts the spooler service start type. It is a pure
// function so the absent-vs-explicit behavior can be unit tested.
func spoolerStartMode(service map[string]registry.RegistryKeyItem) *int64 {
	return regIntPtr(service, "Start")
}

func (r *mqlWindowsSpooler) disabled() (bool, error) {
	reg, err := r.load()
	if err != nil {
		return false, err
	}
	return spoolerDisabled(reg.service), nil
}

// spoolerDisabled reports whether the spooler service start type is Disabled
// (4). An absent start type is treated as not disabled.
func spoolerDisabled(service map[string]registry.RegistryKeyItem) bool {
	if v := regIntPtr(service, "Start"); v != nil {
		return *v == 4
	}
	return false
}

func (r *mqlWindowsSpooler) registerRemoteRpcEndpoint() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.RegisterRemoteRpcEndpoint,
		regIntPtr(reg.printers, "RegisterSpoolerRemoteRpcEndPoint"))
}

func (r *mqlWindowsSpooler) redirectionGuardPolicy() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.RedirectionGuardPolicy,
		regIntPtr(reg.printers, "RedirectionguardPolicy"))
}

func (r *mqlWindowsSpooler) webPnpDownloadDisabled() (bool, error) {
	reg, err := r.load()
	if err != nil {
		return false, err
	}
	return regBoolDefault(reg.printers, "DisableWebPnPDownload", false), nil
}

func (r *mqlWindowsSpooler) httpPrintingDisabled() (bool, error) {
	reg, err := r.load()
	if err != nil {
		return false, err
	}
	return regBoolDefault(reg.printers, "DisableHTTPPrinting", false), nil
}

func (r *mqlWindowsSpooler) copyFilesPolicy() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.CopyFilesPolicy, regIntPtr(reg.printers, "CopyFilesPolicy"))
}

func (r *mqlWindowsSpooler) rpcAuthnLevelPrivacyEnabled() (bool, error) {
	reg, err := r.load()
	if err != nil {
		return false, err
	}
	// Modern Windows enables RPC packet privacy for the spooler by default.
	return regBoolDefault(reg.controlPrint, "RpcAuthnLevelPrivacyEnabled", true), nil
}

func (r *mqlWindowsSpooler) addPrinterDriversRestricted() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.AddPrinterDriversRestricted,
		regIntPtr(reg.lanManServers, "AddPrinterDrivers"))
}

func (r *mqlWindowsSpooler) windowsProtectedPrintGroupPolicyState() (int64, error) {
	reg, err := r.load()
	if err != nil {
		return 0, err
	}
	return resolveNullableInt(&r.WindowsProtectedPrintGroupPolicyState,
		regIntPtr(reg.wpp, "WindowsProtectedPrintGroupPolicyState"))
}

// --- sub-resources ----------------------------------------------------------

func (r *mqlWindowsSpooler) pointAndPrint() (*mqlWindowsSpoolerPointAndPrint, error) {
	reg, err := r.load()
	if err != nil {
		return nil, err
	}
	o, err := CreateResource(r.MqlRuntime, "windows.spooler.pointAndPrint",
		spoolerPointAndPrintArgs(reg.pointAndPrint))
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsSpoolerPointAndPrint), nil
}

// spoolerPointAndPrintArgs builds the args for the PointAndPrint sub-resource.
// restrictDriverInstallationToAdministrators defaults to true — the hardened
// default established by the PrintNightmare mitigations — while the prompt
// settings are nullable so an explicit 0 (keep the prompt) is preserved.
func spoolerPointAndPrintArgs(items map[string]registry.RegistryKeyItem) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id": llx.StringData("windows.spooler.pointAndPrint"),
		"restrictDriverInstallationToAdministrators": llx.BoolData(
			regBoolDefault(items, "RestrictDriverInstallationToAdministrators", true)),
		"noWarningNoElevationOnInstall": llx.IntDataPtr(regIntPtr(items, "NoWarningNoElevationOnInstall")),
		"updatePromptSettings":          llx.IntDataPtr(regIntPtr(items, "UpdatePromptSettings")),
	}
}

func (r *mqlWindowsSpooler) rpc() (*mqlWindowsSpoolerRpc, error) {
	reg, err := r.load()
	if err != nil {
		return nil, err
	}
	o, err := CreateResource(r.MqlRuntime, "windows.spooler.rpc", spoolerRpcArgs(reg.rpc))
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsSpoolerRpc), nil
}

// spoolerRpcArgs builds the args for the RPC sub-resource. All DWORD settings
// are nullable so an unconfigured value (and the compliant 0 of
// useNamedPipeProtocol) is distinguishable from a default.
func spoolerRpcArgs(items map[string]registry.RegistryKeyItem) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":                 llx.StringData("windows.spooler.rpc"),
		"useNamedPipeProtocol": llx.IntDataPtr(regIntPtr(items, "RpcUseNamedPipeProtocol")),
		"authentication":       llx.IntDataPtr(regIntPtr(items, "RpcAuthentication")),
		"protocols":            llx.IntDataPtr(regIntPtr(items, "RpcProtocols")),
		"tcpPort":              llx.IntDataPtr(regIntPtr(items, "RpcTcpPort")),
		"forceKerberos":        llx.BoolData(regBoolDefault(items, "ForceKerberosForRpc", false)),
	}
}

func (r *mqlWindowsSpooler) ipp() (*mqlWindowsSpoolerIpp, error) {
	reg, err := r.load()
	if err != nil {
		return nil, err
	}
	o, err := CreateResource(r.MqlRuntime, "windows.spooler.ipp", spoolerIppArgs(reg.ipp))
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsSpoolerIpp), nil
}

// spoolerIppArgs builds the args for the IPP sub-resource. Each certificate
// check defaults to false (not enforced) when its value is absent.
func spoolerIppArgs(items map[string]registry.RegistryKeyItem) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":                 llx.StringData("windows.spooler.ipp"),
		"requireIpps":          llx.BoolData(regBoolDefault(items, "RequireIpps", false)),
		"blockUnknownCA":       llx.BoolData(regBoolDefault(items, "SecurityFlagsBlockUnknownCA", false)),
		"blockCertWrongUsage":  llx.BoolData(regBoolDefault(items, "SecurityFlagsBlockCertWrongUsage", false)),
		"blockCertDateInvalid": llx.BoolData(regBoolDefault(items, "SecurityFlagsBlockCertDateInvalid", false)),
		"blockCertCNInvalid":   llx.BoolData(regBoolDefault(items, "SecurityFlagsBlockCertCNInvalid", false)),
	}
}
