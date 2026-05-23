// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// FirewallRule is the shared type for cluster, node, and VM firewall rules.
type FirewallRule struct {
	Pos     int    `json:"pos"`
	Type    string `json:"type"`
	Action  string `json:"action"`
	Comment string `json:"comment"`
	Dest    string `json:"dest"`
	Dport   string `json:"dport"`
	Enable  int    `json:"enable"`
	Iface   string `json:"iface"`
	Log     string `json:"log"`
	Macro   string `json:"macro"`
	Proto   string `json:"proto"`
	Source  string `json:"source"`
	Sport   string `json:"sport"`
}

// ---------------------------------------------------------------------------
// Firewall options (cluster, node, VM, container)
// ---------------------------------------------------------------------------

func (c *PveConnection) GetClusterFirewallOptions() (map[string]any, error) {
	var opts map[string]any
	if err := c.apiGet("/cluster/firewall/options", &opts); err != nil {
		return nil, fmt.Errorf("failed to get cluster firewall options: %w", err)
	}
	return opts, nil
}

func (c *PveConnection) GetNodeFirewallOptions(node string) (map[string]any, error) {
	var opts map[string]any
	path := fmt.Sprintf("/nodes/%s/firewall/options", node)
	if err := c.apiGet(path, &opts); err != nil {
		return nil, fmt.Errorf("failed to get firewall options for node %s: %w", node, err)
	}
	return opts, nil
}

func (c *PveConnection) GetVMFirewallOptions(node string, vmid int) (map[string]any, error) {
	var opts map[string]any
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", node, vmid)
	if err := c.apiGet(path, &opts); err != nil {
		return nil, fmt.Errorf("failed to get firewall options for VM %d: %w", vmid, err)
	}
	return opts, nil
}

func (c *PveConnection) GetContainerFirewallOptions(node string, vmid int) (map[string]any, error) {
	var opts map[string]any
	path := fmt.Sprintf("/nodes/%s/lxc/%d/firewall/options", node, vmid)
	if err := c.apiGet(path, &opts); err != nil {
		return nil, fmt.Errorf("failed to get firewall options for container %d: %w", vmid, err)
	}
	return opts, nil
}

// ---------------------------------------------------------------------------
// IPsets — cluster + per-guest
// ---------------------------------------------------------------------------

type IPSetInfo struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

type IPSetEntry struct {
	CIDR    string `json:"cidr"`
	Comment string `json:"comment"`
	NoMatch int    `json:"nomatch"`
}

func (c *PveConnection) GetClusterIPSets() ([]IPSetInfo, error) {
	var sets []IPSetInfo
	if err := c.apiGet("/cluster/firewall/ipset", &sets); err != nil {
		return nil, fmt.Errorf("failed to get cluster ipsets: %w", err)
	}
	return sets, nil
}

func (c *PveConnection) GetClusterIPSetEntries(name string) ([]IPSetEntry, error) {
	var entries []IPSetEntry
	path := fmt.Sprintf("/cluster/firewall/ipset/%s", name)
	if err := c.apiGet(path, &entries); err != nil {
		return nil, fmt.Errorf("failed to get cluster ipset %s entries: %w", name, err)
	}
	return entries, nil
}

func (c *PveConnection) GetVMIPSets(node string, vmid int) ([]IPSetInfo, error) {
	var sets []IPSetInfo
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset", node, vmid)
	if err := c.apiGet(path, &sets); err != nil {
		return nil, fmt.Errorf("failed to get VM %d ipsets: %w", vmid, err)
	}
	return sets, nil
}

func (c *PveConnection) GetVMIPSetEntries(node string, vmid int, name string) ([]IPSetEntry, error) {
	var entries []IPSetEntry
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s", node, vmid, name)
	if err := c.apiGet(path, &entries); err != nil {
		return nil, fmt.Errorf("failed to get VM %d ipset %s entries: %w", vmid, name, err)
	}
	return entries, nil
}

func (c *PveConnection) GetContainerIPSets(node string, vmid int) ([]IPSetInfo, error) {
	var sets []IPSetInfo
	path := fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset", node, vmid)
	if err := c.apiGet(path, &sets); err != nil {
		return nil, fmt.Errorf("failed to get container %d ipsets: %w", vmid, err)
	}
	return sets, nil
}

func (c *PveConnection) GetContainerIPSetEntries(node string, vmid int, name string) ([]IPSetEntry, error) {
	var entries []IPSetEntry
	path := fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s", node, vmid, name)
	if err := c.apiGet(path, &entries); err != nil {
		return nil, fmt.Errorf("failed to get container %d ipset %s entries: %w", vmid, name, err)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Aliases — cluster + per-guest
// ---------------------------------------------------------------------------

type AliasInfo struct {
	Name    string `json:"name"`
	CIDR    string `json:"cidr"`
	Comment string `json:"comment"`
	IPVer   int    `json:"ipversion"`
}

func (c *PveConnection) GetClusterAliases() ([]AliasInfo, error) {
	var aliases []AliasInfo
	if err := c.apiGet("/cluster/firewall/aliases", &aliases); err != nil {
		return nil, fmt.Errorf("failed to get cluster aliases: %w", err)
	}
	return aliases, nil
}

func (c *PveConnection) GetVMAliases(node string, vmid int) ([]AliasInfo, error) {
	var aliases []AliasInfo
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/aliases", node, vmid)
	if err := c.apiGet(path, &aliases); err != nil {
		return nil, fmt.Errorf("failed to get VM %d aliases: %w", vmid, err)
	}
	return aliases, nil
}

func (c *PveConnection) GetContainerAliases(node string, vmid int) ([]AliasInfo, error) {
	var aliases []AliasInfo
	path := fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases", node, vmid)
	if err := c.apiGet(path, &aliases); err != nil {
		return nil, fmt.Errorf("failed to get container %d aliases: %w", vmid, err)
	}
	return aliases, nil
}
