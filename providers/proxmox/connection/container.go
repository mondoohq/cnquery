// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Container listing
// ---------------------------------------------------------------------------

// ContainerInfo describes an LXC container as returned by /cluster/resources.
// The shape mirrors VMInfo, but container-specific config (unprivileged,
// features, hostname) only appears in the per-container config endpoint.
type ContainerInfo struct {
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

func (c *PveConnection) GetAllContainers() ([]ContainerInfo, error) {
	var resources []ContainerInfo
	if err := c.apiGet("/cluster/resources?type=vm", &resources); err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	var containers []ContainerInfo
	for _, r := range resources {
		if r.Type == "lxc" {
			containers = append(containers, r)
		}
	}
	return containers, nil
}

func (c *PveConnection) GetNodeContainers(node string) ([]ContainerInfo, error) {
	all, err := c.GetAllContainers()
	if err != nil {
		return nil, err
	}
	var out []ContainerInfo
	for _, ct := range all {
		if ct.Node == node {
			out = append(out, ct)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Container configuration
// ---------------------------------------------------------------------------

func (c *PveConnection) GetContainerConfig(node string, vmid int) (map[string]any, error) {
	var config map[string]any
	path := fmt.Sprintf("/nodes/%s/lxc/%d/config", node, vmid)
	if err := c.apiGet(path, &config); err != nil {
		return nil, fmt.Errorf("failed to get config for container %d: %w", vmid, err)
	}
	return config, nil
}

// ---------------------------------------------------------------------------
// Container snapshots
// ---------------------------------------------------------------------------

func (c *PveConnection) GetContainerSnapshots(node string, vmid int) ([]SnapshotInfo, error) {
	var snapshots []SnapshotInfo
	path := fmt.Sprintf("/nodes/%s/lxc/%d/snapshot", node, vmid)
	if err := c.apiGet(path, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to get snapshots for container %d: %w", vmid, err)
	}
	var filtered []SnapshotInfo
	for _, s := range snapshots {
		if s.Name != "current" {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

// ---------------------------------------------------------------------------
// Container firewall rules
// ---------------------------------------------------------------------------

func (c *PveConnection) GetContainerFirewallRules(node string, vmid int) ([]FirewallRule, error) {
	var rules []FirewallRule
	path := fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules", node, vmid)
	if err := c.apiGet(path, &rules); err != nil {
		return nil, fmt.Errorf("failed to get firewall rules for container %d: %w", vmid, err)
	}
	return rules, nil
}
