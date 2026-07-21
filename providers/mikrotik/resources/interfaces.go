// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// --- interface ---

func interfaceArgs(row map[string]string) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":             llx.StringData("mikrotik.interface/" + row["name"]),
		"name":             llx.StringData(row["name"]),
		"defaultName":      llx.StringData(row["default-name"]),
		"type":             llx.StringData(row["type"]),
		"mtu":              llx.IntData(parseInt(row["mtu"])),
		"actualMtu":        llx.IntData(parseInt(row["actual-mtu"])),
		"l2mtu":            llx.IntData(parseInt(row["l2mtu"])),
		"maxL2mtu":         llx.IntData(parseInt(row["max-l2mtu"])),
		"macAddress":       llx.StringData(row["mac-address"]),
		"lastLinkUpTime":   llx.StringData(row["last-link-up-time"]),
		"lastLinkDownTime": llx.StringData(row["last-link-down-time"]),
		"linkDowns":        llx.IntData(parseInt(row["link-downs"])),
		"rxByte":           llx.IntData(parseInt(row["rx-byte"])),
		"txByte":           llx.IntData(parseInt(row["tx-byte"])),
		"rxPacket":         llx.IntData(parseInt(row["rx-packet"])),
		"txPacket":         llx.IntData(parseInt(row["tx-packet"])),
		"rxDrop":           llx.IntData(parseInt(row["rx-drop"])),
		"txDrop":           llx.IntData(parseInt(row["tx-drop"])),
		"rxError":          llx.IntData(parseInt(row["rx-error"])),
		"txError":          llx.IntData(parseInt(row["tx-error"])),
		"running":          llx.BoolData(parseBool(row["running"])),
		"slave":            llx.BoolData(parseBool(row["slave"])),
		"dynamic":          llx.BoolData(parseBool(row["dynamic"])),
		"disabled":         llx.BoolData(parseBool(row["disabled"])),
		"comment":          llx.StringData(row["comment"]),
	}
}

func newMikrotikInterface(runtime *plugin.Runtime, row map[string]string) (plugin.Resource, error) {
	return CreateResource(runtime, "mikrotik.interface", interfaceArgs(row))
}

// initMikrotikInterface resolves a mikrotik.interface that was looked up by
// name only (e.g. from an ip.address or vlan cross-reference).
func initMikrotikInterface(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["name"] == nil {
		return args, nil, nil
	}
	name, ok := args["name"].Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}

	rows, err := mikrotikConn(runtime).Print("/interface")
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if row["name"] == name {
			return interfaceArgs(row), nil, nil
		}
	}
	// interface no longer present — keep the bare reference rather than error
	return args, nil, nil
}

func (r *mqlMikrotikInterface) addresses() ([]any, error) {
	conn := mikrotikConn(r.MqlRuntime)
	rows, err := conn.Print("/ip/address")
	if err != nil {
		return nil, err
	}
	name := r.Name.Data
	res := []any{}
	for _, row := range rows {
		if row["interface"] != name && row["actual-interface"] != name {
			continue
		}
		addr, err := newMikrotikIpAddress(r.MqlRuntime, row)
		if err != nil {
			return nil, err
		}
		res = append(res, addr)
	}
	return res, nil
}

func (r *mqlMikrotikInterface) ipv6Addresses() ([]any, error) {
	conn := mikrotikConn(r.MqlRuntime)
	rows, err := conn.Print("/ipv6/address")
	if err != nil {
		return nil, err
	}
	name := r.Name.Data
	res := []any{}
	for _, row := range rows {
		if row["interface"] != name && row["actual-interface"] != name {
			continue
		}
		addr, err := newMikrotikIpv6Address(r.MqlRuntime, row)
		if err != nil {
			return nil, err
		}
		res = append(res, addr)
	}
	return res, nil
}

// --- bridge ---

func newMikrotikBridge(runtime *plugin.Runtime, row map[string]string) (plugin.Resource, error) {
	return CreateResource(runtime, "mikrotik.interface.bridge", map[string]*llx.RawData{
		"__id":          llx.StringData("mikrotik.interface.bridge/" + row["name"]),
		"name":          llx.StringData(row["name"]),
		"macAddress":    llx.StringData(row["mac-address"]),
		"protocolMode":  llx.StringData(row["protocol-mode"]),
		"vlanFiltering": llx.BoolData(parseBool(row["vlan-filtering"])),
		"fastForward":   llx.BoolData(parseBool(row["fast-forward"])),
		"igmpSnooping":  llx.BoolData(parseBool(row["igmp-snooping"])),
		"dhcpSnooping":  llx.BoolData(parseBool(row["dhcp-snooping"])),
		"mtu":           llx.IntData(parseInt(row["mtu"])),
		"actualMtu":     llx.IntData(parseInt(row["actual-mtu"])),
		"l2mtu":         llx.IntData(parseInt(row["l2mtu"])),
		"ageingTime":    llx.StringData(row["ageing-time"]),
		"priority":      llx.StringData(row["priority"]),
		"arp":           llx.StringData(row["arp"]),
		"running":       llx.BoolData(parseBool(row["running"])),
		"disabled":      llx.BoolData(parseBool(row["disabled"])),
		"comment":       llx.StringData(row["comment"]),
	})
}

// --- vlan ---

type mqlMikrotikInterfaceVlanInternal struct {
	cacheInterface string
}

func newMikrotikVlan(runtime *plugin.Runtime, row map[string]string) (plugin.Resource, error) {
	res, err := CreateResource(runtime, "mikrotik.interface.vlan", map[string]*llx.RawData{
		"__id":          llx.StringData("mikrotik.interface.vlan/" + row["name"]),
		"name":          llx.StringData(row["name"]),
		"vlanId":        llx.IntData(parseInt(row["vlan-id"])),
		"mtu":           llx.IntData(parseInt(row["mtu"])),
		"l2mtu":         llx.IntData(parseInt(row["l2mtu"])),
		"macAddress":    llx.StringData(row["mac-address"]),
		"useServiceTag": llx.BoolData(parseBool(row["use-service-tag"])),
		"running":       llx.BoolData(parseBool(row["running"])),
		"disabled":      llx.BoolData(parseBool(row["disabled"])),
		"comment":       llx.StringData(row["comment"]),
	})
	if err != nil {
		return nil, err
	}
	res.(*mqlMikrotikInterfaceVlan).cacheInterface = row["interface"]
	return res, nil
}

func (r *mqlMikrotikInterfaceVlan) compute_interface() (*mqlMikrotikInterface, error) {
	if r.cacheInterface == "" {
		r.Interface.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return interfaceByName(r.MqlRuntime, r.cacheInterface)
}
