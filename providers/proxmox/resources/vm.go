// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

type mqlProxmoxVmInternal struct {
	configFetched bool
	vmConfig      map[string]interface{}
	configErr     error
	osInfoFetched bool
	osInfo        *connection.OsInfo
	osInfoErr     error
	lock          sync.Mutex
}

func vmConn(r *mqlProxmoxVm) *connection.PveConnection {
	return r.MqlRuntime.Connection.(*connection.PveConnection)
}

func (r *mqlProxmoxVm) id() (string, error) {
	return fmt.Sprintf("proxmox.vm/%d", r.Id.Data), nil
}

func (r *mqlProxmoxVm) ensureConfig() {
	if r.configFetched {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.configFetched {
		return
	}
	r.vmConfig, r.configErr = vmConn(r).GetVMConfig(r.Node.Data, int(r.Id.Data))
	r.configFetched = true
}

func (r *mqlProxmoxVm) cfgStr(key string) string {
	r.ensureConfig()
	if r.vmConfig == nil { return "" }
	if v, ok := r.vmConfig[key]; ok { return fmt.Sprintf("%v", v) }
	return ""
}

func (r *mqlProxmoxVm) cfgBool(key string) bool {
	r.ensureConfig()
	if r.vmConfig == nil { return false }
	v, ok := r.vmConfig[key]
	if !ok { return false }
	switch val := v.(type) {
	case bool: return val
	case float64: return val == 1
	case string: return val == "1" || val == "true"
	}
	return false
}

func (r *mqlProxmoxVm) config() (any, error) {
	r.ensureConfig()
	return r.vmConfig, r.configErr
}

func (r *mqlProxmoxVm) osType() (string, error) { return r.cfgStr("ostype"), nil }
func (r *mqlProxmoxVm) machine() (string, error) { return r.cfgStr("machine"), nil }

func (r *mqlProxmoxVm) bios() (string, error) {
	b := r.cfgStr("bios")
	if b == "" { b = "seabios" }
	return b, nil
}

func (r *mqlProxmoxVm) bootOrder() (string, error)   { return r.cfgStr("boot"), nil }
func (r *mqlProxmoxVm) agent() (bool, error)          { return r.cfgBool("agent"), nil }
func (r *mqlProxmoxVm) protection() (bool, error)     { return r.cfgBool("protection"), nil }
func (r *mqlProxmoxVm) description() (string, error)  { return r.cfgStr("description"), nil }

func (r *mqlProxmoxVm) tags() ([]any, error) {
	tagStr := r.cfgStr("tags")
	if tagStr == "" { return []any{}, nil }
	parts := strings.Split(tagStr, ";")
	result := make([]any, len(parts))
	for i, p := range parts { result[i] = p }
	return result, nil
}

func (r *mqlProxmoxVm) networks() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil { return nil, r.configErr }
	var list []any
	for key, val := range r.vmConfig {
		if !strings.HasPrefix(key, "net") { continue }
		valStr := fmt.Sprintf("%v", val)
		net := parseVMNetworkConfig(key, valStr)
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.network", net)
		if err != nil { return nil, err }
		res.(*mqlProxmoxVmNetwork).parentVmid = r.Id.Data
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxVm) disks() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil { return nil, r.configErr }
	prefixes := []string{"scsi", "virtio", "ide", "sata", "efidisk", "tpmstate"}
	var list []any
	for key, val := range r.vmConfig {
		isDisk := false
		for _, p := range prefixes {
			if strings.HasPrefix(key, p) { isDisk = true; break }
		}
		if !isDisk { continue }
		valStr := fmt.Sprintf("%v", val)
		if !strings.Contains(valStr, ":") { continue }
		disk := parseVMDiskConfig(key, valStr)
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.disk", disk)
		if err != nil { return nil, err }
		res.(*mqlProxmoxVmDisk).parentVmid = r.Id.Data
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxVm) snapshots() ([]any, error) {
	conn := vmConn(r)
	snaps, err := conn.GetVMSnapshots(r.Node.Data, int(r.Id.Data))
	if err != nil { return nil, err }
	list := make([]any, len(snaps))
	for i, s := range snaps {
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.snapshot", map[string]*llx.RawData{
			"name":        llx.StringData(s.Name),
			"description": llx.StringData(s.Description),
			"parent":      llx.StringData(s.Parent),
			"snaptime":    llx.IntData(s.Snaptime),
			"vmstate":     llx.BoolData(s.VMState == 1),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxVmSnapshot).parentVmid = r.Id.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxVm) firewallRules() ([]any, error) {
	conn := vmConn(r)
	rules, err := conn.GetVMFirewallRules(r.Node.Data, int(r.Id.Data))
	if err != nil { return nil, err }
	return firewallRulesToResources(r.MqlRuntime, rules)
}

func (r *mqlProxmoxVm) updates() ([]any, error) {
	if r.Status.Data != "running" { return []any{}, nil }
	conn := vmConn(r)
	vmid := int(r.Id.Data)
	if !r.osInfoFetched {
		r.lock.Lock()
		defer r.lock.Unlock()
		if !r.osInfoFetched {
			r.osInfo, r.osInfoErr = conn.GetOsInfo(r.Node.Data, vmid)
			r.osInfoFetched = true
		}
	}
	if r.osInfoErr != nil {
		if errors.Is(r.osInfoErr, connection.ErrQGANotRunning) {
			return nil, fmt.Errorf("guest agent not reachable for VM %d", vmid)
		}
		return nil, r.osInfoErr
	}
	updates, err := conn.GetUpdates(r.Node.Data, vmid, r.osInfo)
	if err != nil { return nil, err }
	list := make([]any, len(updates))
	for i, u := range updates {
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.update", map[string]*llx.RawData{
			"name":             llx.StringData(u.Name),
			"installedVersion": llx.StringData(u.InstalledVersion),
			"newVersion":       llx.StringData(u.NewVersion),
			"upgradable":       llx.BoolData(u.Upgradable),
			"severity":         llx.StringData(u.Severity),
		})
		if err != nil { return nil, err }
		res.(*mqlProxmoxVmUpdate).parentVmid = r.Id.Data
		list[i] = res
	}
	return list, nil
}

// --- User tokens ---
func (r *mqlProxmoxUser) tokens() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PveConnection)
	tokens, err := conn.GetUserTokens(r.Id.Data)
	if err != nil { return nil, err }
	list := make([]any, len(tokens))
	for i, t := range tokens {
		fullID := r.Id.Data + "!" + t.TokenID
		res, err := CreateResource(r.MqlRuntime, "proxmox.token", map[string]*llx.RawData{
			"id":      llx.StringData(fullID),
			"comment": llx.StringData(t.Comment),
			"expire":  llx.IntData(t.Expire),
			"privsep": llx.BoolData(t.Privsep == 1),
		})
		if err != nil { return nil, err }
		list[i] = res
	}
	return list, nil
}
