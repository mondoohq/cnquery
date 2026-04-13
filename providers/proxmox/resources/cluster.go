// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
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

func (r *mqlProxmoxCluster) options() (any, error) {
	conn := clusterConn(r)
	return conn.GetClusterOptions()
}
