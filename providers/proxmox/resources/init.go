// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
	"go.mondoo.com/mql/v13/types"
)

// init functions populate a resource when it was looked up by id only —
// e.g. an ACL entry resolving its target user, group, role, or token.

func initProxmoxUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	userID := args["id"].Value.(string)
	if userID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	users, err := conn.GetUsers()
	if err != nil {
		return nil, nil, err
	}
	for _, u := range users {
		if u.UserID != userID {
			continue
		}
		realm := ""
		if parts := strings.SplitN(u.UserID, "@", 2); len(parts) == 2 {
			realm = parts[1]
		}
		args["email"] = llx.StringData(u.Email)
		args["enable"] = llx.BoolData(u.Enable == 1)
		args["expire"] = llx.IntData(u.Expire)
		args["firstname"] = llx.StringData(u.Firstname)
		args["lastname"] = llx.StringData(u.Lastname)
		args["realm"] = llx.StringData(realm)
		args["realmType"] = llx.StringData(u.RealmType)
		args["tfaLockedUntil"] = llx.IntData(u.TFALockedUntil)
		// Build the resource here rather than returning args: group membership
		// arrives inline on /access/users and groupRefs() reads it from the
		// internal cache, which can only be set on a resource we hold.
		res, err := CreateResource(runtime, "proxmox.user", args)
		if err != nil {
			return nil, nil, err
		}
		mqlUser := res.(*mqlProxmoxUser)
		mqlUser.cachedGroups = splitProxmoxGroups(u.Groups)
		return nil, res, nil
	}
	// User referenced by ACL no longer exists — return the bare resource
	// so audits can still see the dangling reference rather than erroring.
	return args, nil, nil
}

func initProxmoxGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	groupID := args["id"].Value.(string)
	if groupID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	groups, err := conn.GetGroups()
	if err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		if g.GroupID != groupID {
			continue
		}
		var memberIds []any
		if g.Users != "" {
			for _, u := range strings.Split(g.Users, ",") {
				if u = strings.TrimSpace(u); u != "" {
					memberIds = append(memberIds, u)
				}
			}
		}
		args["comment"] = llx.StringData(g.Comment)
		args["memberIds"] = llx.ArrayData(memberIds, "\x02")
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	roleID := args["id"].Value.(string)
	if roleID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	roles, err := conn.GetRoles()
	if err != nil {
		return nil, nil, err
	}
	for _, r := range roles {
		if r.RoleID != roleID {
			continue
		}
		var privs []any
		if r.Privs != "" {
			for _, p := range strings.Split(r.Privs, ",") {
				privs = append(privs, p)
			}
		}
		args["privs"] = llx.ArrayData(privs, "\x02")
		args["special"] = llx.BoolData(r.Special == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxStorage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	storageID := args["id"].Value.(string)
	if storageID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	s, found, err := conn.LookupStorage(storageID)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return args, nil, nil
	}
	var usagePct float64
	if s.UsedFrac > 0 {
		usagePct = s.UsedFrac * 100.0
	} else if s.Total > 0 {
		usagePct = float64(s.Used) / float64(s.Total) * 100.0
	}
	args["type"] = llx.StringData(s.Type)
	args["content"] = llx.StringData(s.Content)
	args["path"] = llx.StringData(s.Path)
	args["nodes"] = llx.StringData(s.Nodes)
	args["enabled"] = llx.BoolData(s.Enabled != 0)
	args["shared"] = llx.BoolData(s.Shared != 0)
	args["total"] = llx.IntData(s.Total)
	args["used"] = llx.IntData(s.Used)
	args["available"] = llx.IntData(s.Avail)
	args["usagePercent"] = llx.FloatData(usagePct)
	args["encrypted"] = llx.BoolData(s.EncryptionKey != "")
	args["encryptionKey"] = llx.StringData(s.EncryptionKey)
	return args, nil, nil
}

func initProxmoxToken(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	tokenID := args["id"].Value.(string)
	if tokenID == "" {
		return args, nil, nil
	}
	// Token IDs are user@realm!tokenid — derive the owner to call the
	// per-user endpoint instead of paging through every user.
	bangIdx := strings.LastIndex(tokenID, "!")
	if bangIdx <= 0 {
		return args, nil, nil
	}
	owner := tokenID[:bangIdx]
	leaf := tokenID[bangIdx+1:]
	conn := runtime.Connection.(*connection.PveConnection)
	tokens, err := conn.GetUserTokens(owner)
	if err != nil {
		return args, nil, nil
	}
	for _, t := range tokens {
		if t.TokenID != leaf {
			continue
		}
		args["comment"] = llx.StringData(t.Comment)
		args["expire"] = llx.IntData(t.Expire)
		args["privsep"] = llx.BoolData(t.Privsep == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxNode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["name"] == nil {
		return args, nil, nil
	}
	name := args["name"].Value.(string)
	if name == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	n, found, err := conn.LookupNode(name)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return args, nil, nil
	}
	args["status"] = llx.StringData(n.Status)
	// The address comes from the cluster status, which a standalone host
	// has no row in. The index leaves it empty in that case.
	if n.IP != "" {
		args["ip"] = llx.StringData(n.IP)
	}
	return args, nil, nil
}

func initProxmoxClusterHaGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	groupID := args["id"].Value.(string)
	if groupID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	groups, err := conn.GetHAGroups()
	if err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		if g.Group != groupID {
			continue
		}
		args["nodes"] = llx.StringData(g.Nodes)
		args["restricted"] = llx.BoolData(g.Restricted == 1)
		args["noFailback"] = llx.BoolData(g.NoFailback == 1)
		args["comment"] = llx.StringData(g.Comment)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxSdnZone(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["zone"] == nil {
		return args, nil, nil
	}
	zoneID := args["zone"].Value.(string)
	if zoneID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	zones, err := conn.GetSDNZones()
	if err != nil {
		return nil, nil, err
	}
	for _, z := range zones {
		if z.Zone != zoneID {
			continue
		}
		args["type"] = llx.StringData(z.Type)
		args["ipam"] = llx.StringData(z.IPAM)
		args["mtu"] = llx.IntData(int64(z.MTU))
		args["nodes"] = llx.StringData(z.Nodes)
		args["dns"] = llx.StringData(z.DNS)
		args["dnsZone"] = llx.StringData(z.DNSZone)
		args["reverseDns"] = llx.StringData(z.ReverseDNS)
		args["pending"] = llx.BoolData(z.Pending == 1)
		args["state"] = llx.StringData(z.State)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxSdnVnet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["vnet"] == nil {
		return args, nil, nil
	}
	vnetID := args["vnet"].Value.(string)
	if vnetID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	vnets, err := conn.GetSDNVNets()
	if err != nil {
		return nil, nil, err
	}
	for _, v := range vnets {
		if v.VNet != vnetID {
			continue
		}
		args["zone"] = llx.StringData(v.Zone)
		args["alias"] = llx.StringData(v.Alias)
		args["tag"] = llx.IntData(int64(v.Tag))
		args["vlanAware"] = llx.BoolData(v.VLANAware == 1)
		args["type"] = llx.StringData(v.Type)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxVm(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	want := args["id"].Value.(int64)
	if want == 0 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	vms, err := conn.GetAllVMs()
	if err != nil {
		return nil, nil, err
	}
	for _, vm := range vms {
		if int64(vm.VMID) != want {
			continue
		}
		args["name"] = llx.StringData(vm.Name)
		args["node"] = llx.StringData(vm.Node)
		args["status"] = llx.StringData(vm.Status)
		args["cpu"] = llx.FloatData(vm.CPU)
		args["maxcpu"] = llx.IntData(int64(vm.MaxCPU))
		args["mem"] = llx.IntData(vm.Mem)
		args["maxmem"] = llx.IntData(vm.MaxMem)
		args["disk"] = llx.IntData(vm.Disk)
		args["maxdisk"] = llx.IntData(vm.MaxDisk)
		args["diskread"] = llx.IntData(vm.DiskRead)
		args["diskwrite"] = llx.IntData(vm.DiskWrite)
		args["netin"] = llx.IntData(vm.NetIn)
		args["netout"] = llx.IntData(vm.NetOut)
		args["uptime"] = llx.IntData(vm.Uptime)
		args["template"] = llx.BoolData(vm.Template == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxContainer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	want := args["id"].Value.(int64)
	if want == 0 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	cts, err := conn.GetAllContainers()
	if err != nil {
		return nil, nil, err
	}
	for _, ct := range cts {
		if int64(ct.VMID) != want {
			continue
		}
		args["name"] = llx.StringData(ct.Name)
		args["node"] = llx.StringData(ct.Node)
		args["status"] = llx.StringData(ct.Status)
		args["cpu"] = llx.FloatData(ct.CPU)
		args["maxcpu"] = llx.IntData(int64(ct.MaxCPU))
		args["mem"] = llx.IntData(ct.Mem)
		args["maxmem"] = llx.IntData(ct.MaxMem)
		args["disk"] = llx.IntData(ct.Disk)
		args["maxdisk"] = llx.IntData(ct.MaxDisk)
		args["diskread"] = llx.IntData(ct.DiskRead)
		args["diskwrite"] = llx.IntData(ct.DiskWrite)
		args["netin"] = llx.IntData(ct.NetIn)
		args["netout"] = llx.IntData(ct.NetOut)
		args["uptime"] = llx.IntData(ct.Uptime)
		args["template"] = llx.BoolData(ct.Template == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxCephFilesystem(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["name"] == nil {
		return args, nil, nil
	}
	name, ok := args["name"].Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	systems, err := conn.GetCephFileSystems()
	if err != nil {
		return nil, nil, err
	}
	for _, fs := range systems {
		if fs.Name != name {
			continue
		}
		args["metadataPool"] = llx.StringData(fs.MetadataPool)
		args["dataPools"] = llx.ArrayData(cephDataPools(fs), types.String)
		return args, nil, nil
	}
	// Reporting a miss as a blank resource would leave metadataPool and
	// dataPools unset, which reads downstream as a file system with no
	// backing pools rather than a lookup that found nothing.
	return nil, nil, fmt.Errorf("proxmox.ceph.filesystem %q not found", name)
}
