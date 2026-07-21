// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/mikrotik/connection"
	"go.mondoo.com/mql/v13/types"
)

func mikrotikConn(runtime *plugin.Runtime) *connection.MikrotikConnection {
	return runtime.Connection.(*connection.MikrotikConnection)
}

// parseInt converts a RouterOS numeric attribute to an int64, returning 0 when
// the value is empty or not a number.
func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseBool converts a RouterOS boolean attribute to a bool. RouterOS reports
// flags as "true"/"false" and occasionally "yes"/"no".
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes":
		return true
	default:
		return false
	}
}

// splitList turns a RouterOS comma-separated attribute (e.g. NTP servers,
// group policies, pool ranges) into a slice suitable for llx.ArrayData,
// trimming whitespace and dropping empty entries.
func splitList(s string) []any {
	if strings.TrimSpace(s) == "" {
		return []any{}
	}
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildList maps each RouterOS reply row to a resource using build.
func buildList(runtime *plugin.Runtime, rows []map[string]string, build func(*plugin.Runtime, map[string]string) (plugin.Resource, error)) ([]any, error) {
	res := make([]any, 0, len(rows))
	for _, row := range rows {
		r, err := build(runtime, row)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// interfaceByName resolves a mikrotik.interface resource by its name.
func interfaceByName(runtime *plugin.Runtime, name string) (*mqlMikrotikInterface, error) {
	res, err := NewResource(runtime, "mikrotik.interface", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMikrotikInterface), nil
}

func (r *mqlMikrotik) id() (string, error) {
	return "mikrotik", nil
}

// --- system family ---

func (r *mqlMikrotik) system() (*mqlMikrotikSystem, error) {
	conn := mikrotikConn(r.MqlRuntime)

	identity, err := conn.PrintOne("/system/identity")
	if err != nil {
		return nil, err
	}
	res, err := conn.PrintOne("/system/resource")
	if err != nil {
		return nil, err
	}
	// routerboard is unavailable on CHR / x86 builds; tolerate failures
	rb, _ := conn.PrintOne("/system/routerboard")

	resource, err := CreateResource(r.MqlRuntime, "mikrotik.system", map[string]*llx.RawData{
		"__id":                 llx.StringData("mikrotik.system"),
		"identity":             llx.StringData(identity["name"]),
		"version":              llx.StringData(res["version"]),
		"buildTime":            llx.StringData(res["build-time"]),
		"factorySoftware":      llx.StringData(res["factory-software"]),
		"boardName":            llx.StringData(res["board-name"]),
		"platform":             llx.StringData(res["platform"]),
		"architecture":         llx.StringData(res["architecture-name"]),
		"cpu":                  llx.StringData(res["cpu"]),
		"cpuCount":             llx.IntData(parseInt(res["cpu-count"])),
		"cpuFrequency":         llx.IntData(parseInt(res["cpu-frequency"])),
		"cpuLoad":              llx.IntData(parseInt(res["cpu-load"])),
		"totalMemory":          llx.IntData(parseInt(res["total-memory"])),
		"freeMemory":           llx.IntData(parseInt(res["free-memory"])),
		"totalHddSpace":        llx.IntData(parseInt(res["total-hdd-space"])),
		"freeHddSpace":         llx.IntData(parseInt(res["free-hdd-space"])),
		"badBlocks":            llx.StringData(res["bad-blocks"]),
		"writeSectSinceReboot": llx.IntData(parseInt(res["write-sect-since-reboot"])),
		"writeSectTotal":       llx.IntData(parseInt(res["write-sect-total"])),
		"uptime":               llx.StringData(res["uptime"]),
		"routerboard":          llx.BoolData(parseBool(rb["routerboard"])),
		"model":                llx.StringData(rb["model"]),
		"serialNumber":         llx.StringData(rb["serial-number"]),
		"firmwareType":         llx.StringData(rb["firmware-type"]),
		"factoryFirmware":      llx.StringData(rb["factory-firmware"]),
		"firmwareVersion":      llx.StringData(rb["current-firmware"]),
		"upgradeFirmware":      llx.StringData(rb["upgrade-firmware"]),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMikrotikSystem), nil
}

func newMikrotikPackage(runtime *plugin.Runtime, row map[string]string) (plugin.Resource, error) {
	return CreateResource(runtime, "mikrotik.system.package", map[string]*llx.RawData{
		"__id":      llx.StringData("mikrotik.system.package/" + row["name"]),
		"name":      llx.StringData(row["name"]),
		"version":   llx.StringData(row["version"]),
		"buildTime": llx.StringData(row["build-time"]),
		"scheduled": llx.StringData(row["scheduled"]),
		"disabled":  llx.BoolData(parseBool(row["disabled"])),
	})
}

func (r *mqlMikrotik) packages() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/system/package")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikPackage)
}

func (r *mqlMikrotik) clock() (*mqlMikrotikSystemClock, error) {
	row, err := mikrotikConn(r.MqlRuntime).PrintOne("/system/clock")
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mikrotik.system.clock", map[string]*llx.RawData{
		"__id":               llx.StringData("mikrotik.system.clock"),
		"time":               llx.StringData(row["time"]),
		"date":               llx.StringData(row["date"]),
		"timeZoneName":       llx.StringData(row["time-zone-name"]),
		"gmtOffset":          llx.StringData(row["gmt-offset"]),
		"dstActive":          llx.BoolData(parseBool(row["dst-active"])),
		"timeZoneAutodetect": llx.BoolData(parseBool(row["time-zone-autodetect"])),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMikrotikSystemClock), nil
}

func (r *mqlMikrotik) ntpClient() (*mqlMikrotikSystemNtpClient, error) {
	row, err := mikrotikConn(r.MqlRuntime).PrintOne("/system/ntp/client")
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mikrotik.system.ntp.client", map[string]*llx.RawData{
		"__id":          llx.StringData("mikrotik.system.ntp.client"),
		"enabled":       llx.BoolData(parseBool(row["enabled"])),
		"mode":          llx.StringData(row["mode"]),
		"servers":       llx.ArrayData(splitList(row["servers"]), types.String),
		"status":        llx.StringData(row["status"]),
		"syncedServer":  llx.StringData(row["synced-server"]),
		"syncedStratum": llx.IntData(parseInt(row["synced-stratum"])),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMikrotikSystemNtpClient), nil
}

func (r *mqlMikrotik) snmp() (*mqlMikrotikSnmp, error) {
	row, err := mikrotikConn(r.MqlRuntime).PrintOne("/snmp")
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mikrotik.snmp", map[string]*llx.RawData{
		"__id":           llx.StringData("mikrotik.snmp"),
		"enabled":        llx.BoolData(parseBool(row["enabled"])),
		"contact":        llx.StringData(row["contact"]),
		"location":       llx.StringData(row["location"]),
		"engineId":       llx.StringData(row["engine-id"]),
		"trapCommunity":  llx.StringData(row["trap-community"]),
		"trapVersion":    llx.IntData(parseInt(row["trap-version"])),
		"trapGenerators": llx.StringData(row["trap-generators"]),
		"srcAddress":     llx.StringData(row["src-address"]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMikrotikSnmp), nil
}

// --- network collections (creators live in interfaces.go / ip.go) ---

func (r *mqlMikrotik) interfaces() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/interface")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikInterface)
}

func (r *mqlMikrotik) bridges() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/interface/bridge")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikBridge)
}

func (r *mqlMikrotik) vlans() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/interface/vlan")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikVlan)
}

func (r *mqlMikrotik) ipAddresses() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/address")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikIpAddress)
}

func (r *mqlMikrotik) ipv6Addresses() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ipv6/address")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikIpv6Address)
}

func (r *mqlMikrotik) routes() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/route")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikRoute)
}

func (r *mqlMikrotik) pools() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/pool")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikIpPool)
}

func (r *mqlMikrotik) dns() (*mqlMikrotikIpDns, error) {
	row, err := mikrotikConn(r.MqlRuntime).PrintOne("/ip/dns")
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mikrotik.ip.dns", map[string]*llx.RawData{
		"__id":                llx.StringData("mikrotik.ip.dns"),
		"servers":             llx.StringData(row["servers"]),
		"dynamicServers":      llx.StringData(row["dynamic-servers"]),
		"allowRemoteRequests": llx.BoolData(parseBool(row["allow-remote-requests"])),
		"useDohServer":        llx.StringData(row["use-doh-server"]),
		"verifyDohCert":       llx.BoolData(parseBool(row["verify-doh-cert"])),
		"maxUdpPacketSize":    llx.IntData(parseInt(row["max-udp-packet-size"])),
		"queryServerTimeout":  llx.StringData(row["query-server-timeout"]),
		"queryTotalTimeout":   llx.StringData(row["query-total-timeout"]),
		"cacheSize":           llx.IntData(parseInt(row["cache-size"])),
		"cacheMaxTtl":         llx.StringData(row["cache-max-ttl"]),
		"cacheUsed":           llx.IntData(parseInt(row["cache-used"])),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMikrotikIpDns), nil
}

func (r *mqlMikrotik) services() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/service")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikService)
}

func (r *mqlMikrotik) firewallRules() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/firewall/filter")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikFirewallFilter)
}

func (r *mqlMikrotik) natRules() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/firewall/nat")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikFirewallNat)
}

func (r *mqlMikrotik) dhcpServers() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/dhcp-server")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikDhcpServer)
}

func (r *mqlMikrotik) dhcpLeases() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/dhcp-server/lease")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikDhcpLease)
}

func (r *mqlMikrotik) neighbors() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/ip/neighbor")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikNeighbor)
}

func (r *mqlMikrotik) users() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/user")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikUser)
}

func (r *mqlMikrotik) userGroups() ([]any, error) {
	rows, err := mikrotikConn(r.MqlRuntime).Print("/user/group")
	if err != nil {
		return nil, err
	}
	return buildList(r.MqlRuntime, rows, newMikrotikUserGroup)
}
