// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

type mqlProxmoxVmInternal struct {
	configFetched bool
	vmConfig      map[string]interface{}
	configErr     error
	osInfoFetched bool
	osInfo        *connection.OsInfo
	osInfoErr     error
	cfgLock       sync.Mutex
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
	r.cfgLock.Lock()
	defer r.cfgLock.Unlock()
	if r.configFetched {
		return
	}
	r.vmConfig, r.configErr = vmConn(r).GetVMConfig(r.Node.Data, int(r.Id.Data))
	r.configFetched = true
}

func (r *mqlProxmoxVm) cfgStr(key string) string {
	r.ensureConfig()
	if r.vmConfig == nil {
		return ""
	}
	if v, ok := r.vmConfig[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (r *mqlProxmoxVm) cfgBool(key string) bool {
	r.ensureConfig()
	if r.vmConfig == nil {
		return false
	}
	v, ok := r.vmConfig[key]
	if !ok {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val == 1
	case string:
		return val == "1" || val == "true"
	}
	return false
}

func (r *mqlProxmoxVm) config() (any, error) {
	r.ensureConfig()
	return r.vmConfig, r.configErr
}

func (r *mqlProxmoxVm) osType() (string, error)  { return r.cfgStr("ostype"), nil }
func (r *mqlProxmoxVm) machine() (string, error) { return r.cfgStr("machine"), nil }

func (r *mqlProxmoxVm) bios() (string, error) {
	b := r.cfgStr("bios")
	if b == "" {
		b = "seabios"
	}
	return b, nil
}

func (r *mqlProxmoxVm) bootOrder() (string, error)   { return r.cfgStr("boot"), nil }
func (r *mqlProxmoxVm) agent() (bool, error)         { return r.cfgBool("agent"), nil }
func (r *mqlProxmoxVm) protection() (bool, error)    { return r.cfgBool("protection"), nil }
func (r *mqlProxmoxVm) description() (string, error) { return r.cfgStr("description"), nil }
func (r *mqlProxmoxVm) lock() (string, error)        { return r.cfgStr("lock"), nil }
func (r *mqlProxmoxVm) hookscript() (string, error)  { return r.cfgStr("hookscript"), nil }
func (r *mqlProxmoxVm) args() (string, error)        { return r.cfgStr("args"), nil }
func (r *mqlProxmoxVm) vga() (string, error)         { return r.cfgStr("vga"), nil }

// --- Cloud-init ---

func (r *mqlProxmoxVm) ciuser() (string, error)       { return r.cfgStr("ciuser"), nil }
func (r *mqlProxmoxVm) sshkeys() (string, error)      { return r.cfgStr("sshkeys"), nil }
func (r *mqlProxmoxVm) searchDomain() (string, error) { return r.cfgStr("searchdomain"), nil }
func (r *mqlProxmoxVm) nameserver() (string, error)   { return r.cfgStr("nameserver"), nil }

// cipasswordSet only reports whether the key is present in the VM
// config; the password value itself is intentionally never read or
// surfaced through this resource. Audits should focus on whether a
// password is configured, not on what it is.
func (r *mqlProxmoxVm) cipasswordSet() (bool, error) {
	r.ensureConfig()
	if r.vmConfig == nil {
		return false, nil
	}
	v, ok := r.vmConfig["cipassword"]
	if !ok {
		return false, nil
	}
	// PVE returns the raw value; a non-empty string means a password is set.
	switch val := v.(type) {
	case string:
		return val != "", nil
	default:
		return v != nil, nil
	}
}

func (r *mqlProxmoxVm) ciCustom() (any, error) {
	// `cicustom` is serialized as comma-delimited key=storage:snippets/file
	// pairs (e.g. `user=local:snippets/u.yaml,network=local:snippets/n.yaml`).
	val := r.cfgStr("cicustom")
	out := map[string]any{}
	if val == "" {
		return out, nil
	}
	for _, part := range strings.Split(val, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out, nil
}

func (r *mqlProxmoxVm) serialPorts() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil {
		return nil, r.configErr
	}
	var list []any
	for key, val := range r.vmConfig {
		if !isSerialPortKey(key) {
			continue
		}
		target := fmt.Sprintf("%v", val)
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.serialPort", map[string]*llx.RawData{
			"__id":   llx.StringData(fmt.Sprintf("proxmox.vm.serialPort/%d/%s", r.Id.Data, key)),
			"id":     llx.StringData(key),
			"target": llx.StringData(target),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

// isSerialPortKey matches serial0..serial3 — the only valid slot names
// in PVE. Anything else (`serialport` style typos, future slots) is
// ignored so the audit sees exactly the ports PVE actually exposes.
func isSerialPortKey(key string) bool {
	if !strings.HasPrefix(key, "serial") {
		return false
	}
	suffix := key[len("serial"):]
	if len(suffix) != 1 {
		return false
	}
	return suffix[0] >= '0' && suffix[0] <= '3'
}

func (r *mqlProxmoxVm) tags() ([]any, error) {
	tagStr := r.cfgStr("tags")
	if tagStr == "" {
		return []any{}, nil
	}
	parts := strings.Split(tagStr, ";")
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result, nil
}

func (r *mqlProxmoxVm) pool() (*mqlProxmoxPool, error) {
	id := r.cfgStr("pool")
	if id == "" {
		r.Pool.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "proxmox.pool", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxPool), nil
}

func (r *mqlProxmoxVm) networks() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil {
		return nil, r.configErr
	}
	var list []any
	for key, val := range r.vmConfig {
		if !strings.HasPrefix(key, "net") {
			continue
		}
		valStr := fmt.Sprintf("%v", val)
		net := parseVMNetworkConfig(key, valStr)
		net["__id"] = llx.StringData(fmt.Sprintf("proxmox.vm.network/%d/%s", r.Id.Data, key))
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.network", net)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxVm) disks() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil {
		return nil, r.configErr
	}
	prefixes := []string{"scsi", "virtio", "ide", "sata", "efidisk", "tpmstate"}
	var list []any
	for key, val := range r.vmConfig {
		isDisk := false
		for _, p := range prefixes {
			if strings.HasPrefix(key, p) {
				isDisk = true
				break
			}
		}
		if !isDisk {
			continue
		}
		valStr := fmt.Sprintf("%v", val)
		if !strings.Contains(valStr, ":") {
			continue
		}
		disk := parseVMDiskConfig(key, valStr)
		disk["__id"] = llx.StringData(fmt.Sprintf("proxmox.vm.disk/%d/%s", r.Id.Data, key))
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.disk", disk)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxVm) snapshots() ([]any, error) {
	conn := vmConn(r)
	snaps, err := conn.GetVMSnapshots(r.Node.Data, int(r.Id.Data))
	if err != nil {
		return nil, err
	}
	list := make([]any, len(snaps))
	for i, s := range snaps {
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.snapshot", map[string]*llx.RawData{
			"__id":        llx.StringData(fmt.Sprintf("proxmox.vm.snapshot/vm/%d/%s", r.Id.Data, s.Name)),
			"name":        llx.StringData(s.Name),
			"description": llx.StringData(s.Description),
			"parent":      llx.StringData(s.Parent),
			"snaptime":    llx.IntData(s.Snaptime),
			"vmstate":     llx.BoolData(s.VMState == 1),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxVm) firewallRules() ([]any, error) {
	conn := vmConn(r)
	rules, err := conn.GetVMFirewallRules(r.Node.Data, int(r.Id.Data))
	if err != nil {
		return nil, err
	}
	return firewallRulesToResources(r.MqlRuntime, rules, fmt.Sprintf("vm/%d", r.Id.Data))
}

func (r *mqlProxmoxVm) updates() ([]any, error) {
	if r.Status.Data != "running" {
		return []any{}, nil
	}
	conn := vmConn(r)
	vmid := int(r.Id.Data)
	if !r.osInfoFetched {
		r.cfgLock.Lock()
		defer r.cfgLock.Unlock()
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
	if err != nil {
		return nil, err
	}
	list := make([]any, len(updates))
	for i, u := range updates {
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.update", map[string]*llx.RawData{
			"__id":             llx.StringData(fmt.Sprintf("proxmox.vm.update/%d/%s", r.Id.Data, u.Name)),
			"name":             llx.StringData(u.Name),
			"installedVersion": llx.StringData(u.InstalledVersion),
			"newVersion":       llx.StringData(u.NewVersion),
			"upgradable":       llx.BoolData(u.Upgradable),
			"severity":         llx.StringData(u.Severity),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

// --- User tokens ---
func (r *mqlProxmoxUser) tokens() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PveConnection)
	tokens, err := conn.GetUserTokens(r.Id.Data)
	if err != nil {
		return nil, err
	}
	list := make([]any, len(tokens))
	for i, t := range tokens {
		fullID := r.Id.Data + "!" + t.TokenID
		res, err := CreateResource(r.MqlRuntime, "proxmox.token", map[string]*llx.RawData{
			"id":      llx.StringData(fullID),
			"comment": llx.StringData(t.Comment),
			"expire":  llx.IntData(t.Expire),
			"privsep": llx.BoolData(t.Privsep == 1),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}
