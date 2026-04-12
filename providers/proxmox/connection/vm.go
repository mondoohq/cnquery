// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

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

// GetNodeVMs returns VMs on a specific node.
func (c *PveConnection) GetNodeVMs(node string) ([]VMInfo, error) {
	allVMs, err := c.GetAllVMs()
	if err != nil {
		return nil, err
	}
	var vms []VMInfo
	for _, vm := range allVMs {
		if vm.Node == node {
			vms = append(vms, vm)
		}
	}
	return vms, nil
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
