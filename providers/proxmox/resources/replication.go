// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlProxmox) replicationJobs() ([]any, error) {
	conn := proxmoxConn(r)
	jobs, err := conn.GetReplicationJobs()
	if err != nil {
		return nil, err
	}
	list := make([]any, len(jobs))
	for i, j := range jobs {
		res, err := CreateResource(r.MqlRuntime, "proxmox.replication.job", map[string]*llx.RawData{
			"id":        llx.StringData(j.ID),
			"vmid":      llx.IntData(int64(j.VMID)),
			"schedule":  llx.StringData(j.Schedule),
			"source":    llx.StringData(j.Source),
			"target":    llx.StringData(j.Target),
			"type":      llx.StringData(j.Type),
			"comment":   llx.StringData(j.Comment),
			"rate":      llx.IntData(int64(j.Rate)),
			"disabled":  llx.BoolData(j.Disable == 1),
			"removeJob": llx.StringData(j.RemoveJob),
		})
		if err != nil {
			return nil, err
		}
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxReplicationJob) sourceNode() (*mqlProxmoxNode, error) {
	return replicationNodeRef(r.MqlRuntime, r.Source.Data, &r.SourceNode)
}

func (r *mqlProxmoxReplicationJob) targetNode() (*mqlProxmoxNode, error) {
	return replicationNodeRef(r.MqlRuntime, r.Target.Data, &r.TargetNode)
}

func replicationNodeRef(runtime *plugin.Runtime, name string, slot *plugin.TValue[*mqlProxmoxNode]) (*mqlProxmoxNode, error) {
	if name == "" {
		slot.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "proxmox.node", map[string]*llx.RawData{
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxNode), nil
}

// vm() and container() share the vmid — only one resolves successfully for
// any given guest. The init functions on proxmox.vm and proxmox.container
// (see init.go) populate the other fields when the resource is referenced
// solely by id.

func (r *mqlProxmoxReplicationJob) vm() (*mqlProxmoxVm, error) {
	res, err := NewResource(r.MqlRuntime, "proxmox.vm", map[string]*llx.RawData{
		"id": llx.IntData(r.Vmid.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxVm), nil
}

func (r *mqlProxmoxReplicationJob) container() (*mqlProxmoxContainer, error) {
	res, err := NewResource(r.MqlRuntime, "proxmox.container", map[string]*llx.RawData{
		"id": llx.IntData(r.Vmid.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxContainer), nil
}
