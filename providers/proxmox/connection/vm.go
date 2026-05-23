// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"net/url"
	"strconv"
)

// ---------------------------------------------------------------------------
// VM listing
// ---------------------------------------------------------------------------

type VMInfo struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Node      string  `json:"node"`
	Status    string  `json:"status"`
	Type      string  `json:"type"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"maxcpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	Uptime    int64   `json:"uptime"`
	Template  int     `json:"template"`
	Tags      string  `json:"tags"`
}

func (c *PveConnection) GetAllVMs() ([]VMInfo, error) {
	var resources []VMInfo
	if err := c.apiGet("/cluster/resources?type=vm", &resources); err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}
	var vms []VMInfo
	for _, r := range resources {
		if r.Type == "qemu" {
			vms = append(vms, r)
		}
	}
	return vms, nil
}

// nodeQemuEntry mirrors a /nodes/<node>/qemu row. VMIDs come back as
// strings on this endpoint (unlike /cluster/resources, which uses
// ints), so we unmarshal into an intermediate type and convert.
type nodeQemuEntry struct {
	VMID      string  `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"cpus"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	Uptime    int64   `json:"uptime"`
	Template  int     `json:"template"`
	Tags      string  `json:"tags"`
}

// GetNodeVMs hits the per-node /nodes/<node>/qemu endpoint directly so
// `proxmox.nodes { vms }` doesn't fan out into one full
// cluster-resources fetch per node.
func (c *PveConnection) GetNodeVMs(node string) ([]VMInfo, error) {
	var entries []nodeQemuEntry
	path := fmt.Sprintf("/nodes/%s/qemu", url.PathEscape(node))
	if err := c.apiGet(path, &entries); err != nil {
		return nil, fmt.Errorf("failed to list VMs on node %s: %w", node, err)
	}
	out := make([]VMInfo, 0, len(entries))
	for _, e := range entries {
		vmid, err := strconv.Atoi(e.VMID)
		if err != nil {
			return nil, fmt.Errorf("invalid VMID %q on node %s: %w", e.VMID, node, err)
		}
		out = append(out, VMInfo{
			VMID:      vmid,
			Name:      e.Name,
			Node:      node,
			Status:    e.Status,
			Type:      "qemu",
			CPU:       e.CPU,
			MaxCPU:    e.MaxCPU,
			Mem:       e.Mem,
			MaxMem:    e.MaxMem,
			Disk:      e.Disk,
			MaxDisk:   e.MaxDisk,
			DiskRead:  e.DiskRead,
			DiskWrite: e.DiskWrite,
			NetIn:     e.NetIn,
			NetOut:    e.NetOut,
			Uptime:    e.Uptime,
			Template:  e.Template,
			Tags:      e.Tags,
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// VM configuration
// ---------------------------------------------------------------------------

func (c *PveConnection) GetVMConfig(node string, vmid int) (map[string]interface{}, error) {
	var config map[string]interface{}
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", node, vmid)
	if err := c.apiGet(path, &config); err != nil {
		return nil, fmt.Errorf("failed to get config for VM %d: %w", vmid, err)
	}
	return config, nil
}

// ---------------------------------------------------------------------------
// VM snapshots
// ---------------------------------------------------------------------------

type SnapshotInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      string `json:"parent"`
	Snaptime    int64  `json:"snaptime"`
	VMState     int    `json:"vmstate"`
}

func (c *PveConnection) GetVMSnapshots(node string, vmid int) ([]SnapshotInfo, error) {
	var snapshots []SnapshotInfo
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", node, vmid)
	if err := c.apiGet(path, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to get snapshots for VM %d: %w", vmid, err)
	}
	// Filter out the "current" pseudo-snapshot
	var filtered []SnapshotInfo
	for _, s := range snapshots {
		if s.Name != "current" {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

// ---------------------------------------------------------------------------
// VM firewall rules
// ---------------------------------------------------------------------------

func (c *PveConnection) GetVMFirewallRules(node string, vmid int) ([]FirewallRule, error) {
	var rules []FirewallRule
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", node, vmid)
	if err := c.apiGet(path, &rules); err != nil {
		return nil, fmt.Errorf("failed to get firewall rules for VM %d: %w", vmid, err)
	}
	return rules, nil
}
