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

// Registry locations that back the windows.tcpip network-hardening settings.
// These are read directly from the registry, so the resource is available on
// connections that cannot run commands.
const (
	tcpipParametersPath  = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
	tcpip6ParametersPath = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters`
	netbtParametersPath  = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\NetBT\Parameters`
)

func (r *mqlWindowsTcpip) id() (string, error) {
	return "windows.tcpip", nil
}

func (r *mqlWindowsTcpipIpv6) id() (string, error) {
	return "windows.tcpip.ipv6", nil
}

func (r *mqlWindowsTcpipNetbios) id() (string, error) {
	return "windows.tcpip.netbios", nil
}

// readTcpipKey reads a single registry key and returns its numeric DWORD values
// keyed by the lower-cased value name. A missing key yields an empty map rather
// than an error, so each field resolves to null (the value is absent). A read
// failure (e.g. no registry access on this connection) surfaces as an error.
func readTcpipKey(runtime *plugin.Runtime, path string) (map[string]int64, error) {
	o, err := CreateResource(runtime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		// a missing key is expected (the values are not configured); treat it as
		// empty so every field resolves to null
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return map[string]int64{}, nil
		}
		return nil, err
	}

	return tcpipItemsToMap(entries), nil
}

// tcpipItemsToMap reduces registry entries to a lower-cased name->number map.
// Split out as a pure function so the case-insensitive lookup can be unit
// tested without a live registry.
func tcpipItemsToMap(entries []registry.RegistryKeyItem) map[string]int64 {
	res := make(map[string]int64, len(entries))
	for i := range entries {
		res[strings.ToLower(entries[i].Key)] = entries[i].Value.Number
	}
	return res
}

// tcpipIntPtr returns a pointer to the DWORD value stored under name (lookup is
// case-insensitive), or nil when the value is absent. Returning nil lets the
// caller emit a null field so an explicit 0 is distinguishable from an
// unconfigured value. Pure function for unit testing.
func tcpipIntPtr(items map[string]int64, name string) *int64 {
	if v, ok := items[strings.ToLower(name)]; ok {
		return &v
	}
	return nil
}

// initWindowsTcpip reads the Tcpip\Parameters key once and populates the
// top-level DWORD fields. Each field is null when its value is absent so an
// explicit 0 is distinguishable from an unconfigured value.
func initWindowsTcpip(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	items, err := readTcpipKey(runtime, tcpipParametersPath)
	if err != nil {
		return nil, nil, err
	}

	args["disableIpSourceRouting"] = llx.IntDataPtr(tcpipIntPtr(items, "DisableIPSourceRouting"))
	args["enableIcmpRedirect"] = llx.IntDataPtr(tcpipIntPtr(items, "EnableICMPRedirect"))
	args["keepAliveTime"] = llx.IntDataPtr(tcpipIntPtr(items, "KeepAliveTime"))
	args["performRouterDiscovery"] = llx.IntDataPtr(tcpipIntPtr(items, "PerformRouterDiscovery"))
	// the TcpMaxDataRetransmissions value name is stored lower-cased on some
	// systems; the lookup is case-insensitive so either casing resolves.
	args["tcpMaxDataRetransmissions"] = llx.IntDataPtr(tcpipIntPtr(items, "TcpMaxDataRetransmissions"))

	return args, nil, nil
}

func (r *mqlWindowsTcpip) ipv6() (*mqlWindowsTcpipIpv6, error) {
	items, err := readTcpipKey(r.MqlRuntime, tcpip6ParametersPath)
	if err != nil {
		return nil, err
	}

	o, err := CreateResource(r.MqlRuntime, "windows.tcpip.ipv6", map[string]*llx.RawData{
		"__id":                   llx.StringData("windows.tcpip.ipv6"),
		"disableIpSourceRouting": llx.IntDataPtr(tcpipIntPtr(items, "DisableIPSourceRouting")),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsTcpipIpv6), nil
}

func (r *mqlWindowsTcpip) netbios() (*mqlWindowsTcpipNetbios, error) {
	items, err := readTcpipKey(r.MqlRuntime, netbtParametersPath)
	if err != nil {
		return nil, err
	}

	o, err := CreateResource(r.MqlRuntime, "windows.tcpip.netbios", map[string]*llx.RawData{
		"__id":                  llx.StringData("windows.tcpip.netbios"),
		"nodeType":              llx.IntDataPtr(tcpipIntPtr(items, "NodeType")),
		"noNameReleaseOnDemand": llx.IntDataPtr(tcpipIntPtr(items, "NoNameReleaseOnDemand")),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlWindowsTcpipNetbios), nil
}
