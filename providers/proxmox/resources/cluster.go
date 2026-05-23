// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

func clusterConn(r *mqlProxmoxCluster) *connection.PveConnection {
	return r.MqlRuntime.Connection.(*connection.PveConnection)
}

func (r *mqlProxmoxCluster) id() (string, error) {
	return "proxmox.cluster/" + r.Name.Data, nil
}

func (r *mqlProxmoxCluster) haResources() ([]any, error) {
	conn := clusterConn(r)
	ha, err := conn.GetHAResources()
	if err != nil {
		return nil, err
	}
	list := make([]any, len(ha))
	for i, h := range ha {
		res, err := CreateResource(r.MqlRuntime, "proxmox.cluster.haResource", map[string]*llx.RawData{
			"id":          llx.StringData(h.SID),
			"type":        llx.StringData(h.Type),
			"status":      llx.StringData(h.Status),
			"node":        llx.StringData(h.Node),
			"maxRestart":  llx.IntData(int64(h.MaxRestart)),
			"maxRelocate": llx.IntData(int64(h.MaxRelocate)),
			"state":       llx.StringData(h.State),
			"group":       llx.StringData(h.Group),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxCluster) firewallRules() ([]any, error) {
	conn := clusterConn(r)
	rules, err := conn.GetClusterFirewallRules()
	if err != nil {
		return nil, err
	}
	return firewallRulesToResources(r.MqlRuntime, rules, "cluster")
}

func (r *mqlProxmoxCluster) haGroups() ([]any, error) {
	conn := clusterConn(r)
	groups, err := conn.GetHAGroups()
	if err != nil {
		return nil, err
	}
	list := make([]any, len(groups))
	for i, g := range groups {
		res, err := CreateResource(r.MqlRuntime, "proxmox.cluster.haGroup", map[string]*llx.RawData{
			"id":         llx.StringData(g.Group),
			"nodes":      llx.StringData(g.Nodes),
			"restricted": llx.BoolData(g.Restricted == 1),
			"noFailback": llx.BoolData(g.NoFailback == 1),
			"comment":    llx.StringData(g.Comment),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxCluster) options() (any, error) {
	conn := clusterConn(r)
	return conn.GetClusterOptions()
}

// ---------------------------------------------------------------------------
// haResource cross-references
// ---------------------------------------------------------------------------

// parseHAResourceSID parses ids like "vm:100" or "ct:200" into their pieces.
// HA configs occasionally drop the prefix and just carry the numeric VMID,
// in which case `kind` is the resource's `type` and `id` is the parsed int.
func parseHAResourceSID(sid, typ string) (kind string, id int64, ok bool) {
	if colon := strings.Index(sid, ":"); colon > 0 {
		kind = sid[:colon]
		parsed, err := strconv.ParseInt(sid[colon+1:], 10, 64)
		if err != nil {
			return "", 0, false
		}
		return kind, parsed, true
	}
	parsed, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		return "", 0, false
	}
	return typ, parsed, true
}

func (r *mqlProxmoxClusterHaResource) vm() (*mqlProxmoxVm, error) {
	kind, id, ok := parseHAResourceSID(r.Id.Data, r.Type.Data)
	if !ok || kind != "vm" {
		r.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "proxmox.vm", map[string]*llx.RawData{
		"id": llx.IntData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxVm), nil
}

func (r *mqlProxmoxClusterHaResource) container() (*mqlProxmoxContainer, error) {
	kind, id, ok := parseHAResourceSID(r.Id.Data, r.Type.Data)
	if !ok || kind != "ct" {
		r.Container.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "proxmox.container", map[string]*llx.RawData{
		"id": llx.IntData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxContainer), nil
}

func (r *mqlProxmoxClusterHaResource) groupRef() (*mqlProxmoxClusterHaGroup, error) {
	if r.Group.Data == "" {
		r.GroupRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "proxmox.cluster.haGroup", map[string]*llx.RawData{
		"id": llx.StringData(r.Group.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxClusterHaGroup), nil
}
