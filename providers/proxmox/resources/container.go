// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

type mqlProxmoxContainerInternal struct {
	configFetched bool
	ctConfig      map[string]any
	configErr     error
	lock          sync.Mutex
}

func ctConn(r *mqlProxmoxContainer) *connection.PveConnection {
	return r.MqlRuntime.Connection.(*connection.PveConnection)
}

func (r *mqlProxmoxContainer) id() (string, error) {
	return fmt.Sprintf("proxmox.container/%d", r.Id.Data), nil
}

func containerInfoToResources(runtime *plugin.Runtime, cts []connection.ContainerInfo) ([]any, error) {
	list := make([]any, len(cts))
	for i, ct := range cts {
		res, err := CreateResource(runtime, "proxmox.container", map[string]*llx.RawData{
			"id":        llx.IntData(int64(ct.VMID)),
			"name":      llx.StringData(ct.Name),
			"node":      llx.StringData(ct.Node),
			"status":    llx.StringData(ct.Status),
			"cpu":       llx.FloatData(ct.CPU),
			"maxcpu":    llx.IntData(int64(ct.MaxCPU)),
			"mem":       llx.IntData(ct.Mem),
			"maxmem":    llx.IntData(ct.MaxMem),
			"disk":      llx.IntData(ct.Disk),
			"maxdisk":   llx.IntData(ct.MaxDisk),
			"diskread":  llx.IntData(ct.DiskRead),
			"diskwrite": llx.IntData(ct.DiskWrite),
			"netin":     llx.IntData(ct.NetIn),
			"netout":    llx.IntData(ct.NetOut),
			"uptime":    llx.IntData(ct.Uptime),
			"template":  llx.BoolData(ct.Template == 1),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmox) containers() ([]any, error) {
	conn := proxmoxConn(r)
	cts, err := conn.GetAllContainers()
	if err != nil {
		return nil, err
	}
	return containerInfoToResources(r.MqlRuntime, cts)
}

func (r *mqlProxmoxNode) containers() ([]any, error) {
	conn := nodeConn(r)
	cts, err := conn.GetNodeContainers(r.Name.Data)
	if err != nil {
		return nil, err
	}
	return containerInfoToResources(r.MqlRuntime, cts)
}

func (r *mqlProxmoxContainer) ensureConfig() {
	if r.configFetched {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.configFetched {
		return
	}
	r.ctConfig, r.configErr = ctConn(r).GetContainerConfig(r.Node.Data, int(r.Id.Data))
	r.configFetched = true
}

func (r *mqlProxmoxContainer) cfgStr(key string) string {
	r.ensureConfig()
	if r.ctConfig == nil {
		return ""
	}
	if v, ok := r.ctConfig[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (r *mqlProxmoxContainer) cfgBool(key string) bool {
	r.ensureConfig()
	if r.ctConfig == nil {
		return false
	}
	v, ok := r.ctConfig[key]
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

func (r *mqlProxmoxContainer) config() (any, error) {
	r.ensureConfig()
	return r.ctConfig, r.configErr
}

func (r *mqlProxmoxContainer) osType() (string, error)      { return r.cfgStr("ostype"), nil }
func (r *mqlProxmoxContainer) hostname() (string, error)    { return r.cfgStr("hostname"), nil }
func (r *mqlProxmoxContainer) unprivileged() (bool, error)  { return r.cfgBool("unprivileged"), nil }
func (r *mqlProxmoxContainer) protection() (bool, error)    { return r.cfgBool("protection"), nil }
func (r *mqlProxmoxContainer) onboot() (bool, error)        { return r.cfgBool("onboot"), nil }
func (r *mqlProxmoxContainer) description() (string, error) { return r.cfgStr("description"), nil }

func (r *mqlProxmoxContainer) features() ([]any, error) {
	val := r.cfgStr("features")
	if val == "" {
		return []any{}, nil
	}
	// features line is comma-separated; an enabled flag is written as
	// `keyctl=1` or `nesting=1` and a parameter-style flag like
	// `mount=nfs;cifs`. We surface the keys whose value isn't "0" since
	// "feature is enabled" is what audits care about.
	var out []any
	for _, part := range strings.Split(val, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if key == "" || v == "0" {
			continue
		}
		out = append(out, key)
	}
	return out, nil
}

func (r *mqlProxmoxContainer) tags() ([]any, error) {
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

func (r *mqlProxmoxContainer) networks() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil {
		return nil, r.configErr
	}
	var list []any
	for key, val := range r.ctConfig {
		if !strings.HasPrefix(key, "net") {
			continue
		}
		valStr := fmt.Sprintf("%v", val)
		net := parseContainerNetworkConfig(key, valStr)
		res, err := CreateResource(r.MqlRuntime, "proxmox.container.network", net)
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxContainerNetwork).parentVmid = r.Id.Data
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxContainer) mountPoints() ([]any, error) {
	r.ensureConfig()
	if r.configErr != nil {
		return nil, r.configErr
	}
	var list []any
	// rootfs is the root mount; mp0..mp254 are extra mounts.
	for key, val := range r.ctConfig {
		if key != "rootfs" && !strings.HasPrefix(key, "mp") {
			continue
		}
		if strings.HasPrefix(key, "mp") {
			// Exclude memory pool keys (mpregion etc. don't exist but be defensive)
			rest := key[2:]
			if rest == "" {
				continue
			}
			allDigit := true
			for _, c := range rest {
				if c < '0' || c > '9' {
					allDigit = false
					break
				}
			}
			if !allDigit {
				continue
			}
		}
		valStr := fmt.Sprintf("%v", val)
		mp := parseContainerMountPoint(key, valStr)
		res, err := CreateResource(r.MqlRuntime, "proxmox.container.mountPoint", mp)
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxContainerMountPoint).parentVmid = r.Id.Data
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlProxmoxContainer) snapshots() ([]any, error) {
	conn := ctConn(r)
	snaps, err := conn.GetContainerSnapshots(r.Node.Data, int(r.Id.Data))
	if err != nil {
		return nil, err
	}
	list := make([]any, len(snaps))
	for i, s := range snaps {
		res, err := CreateResource(r.MqlRuntime, "proxmox.vm.snapshot", map[string]*llx.RawData{
			"name":        llx.StringData(s.Name),
			"description": llx.StringData(s.Description),
			"parent":      llx.StringData(s.Parent),
			"snaptime":    llx.IntData(s.Snaptime),
			"vmstate":     llx.BoolData(s.VMState == 1),
		})
		if err != nil {
			return nil, err
		}
		// Container snapshots share the proxmox.vm.snapshot type, so
		// reuse the same internal parentVmid for cache-key isolation.
		res.(*mqlProxmoxVmSnapshot).parentVmid = r.Id.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxContainer) firewallRules() ([]any, error) {
	conn := ctConn(r)
	rules, err := conn.GetContainerFirewallRules(r.Node.Data, int(r.Id.Data))
	if err != nil {
		return nil, err
	}
	return firewallRulesToResources(r.MqlRuntime, rules, fmt.Sprintf("ct/%d", r.Id.Data))
}
