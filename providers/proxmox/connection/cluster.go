// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Cluster status
// ---------------------------------------------------------------------------

type ClusterStatusEntry struct {
	Type    string `json:"type"` // "cluster" or "node"
	Name    string `json:"name"`
	ID      string `json:"id"`
	Version int    `json:"version"`
	Quorate int    `json:"quorate"`
	Nodes   int    `json:"nodes"`
	// Node-level fields
	NodeID int    `json:"nodeid"`
	IP     string `json:"ip"`
	Online int    `json:"online"`
	Local  int    `json:"local"`
	Level  string `json:"level"`
}

func (c *PveConnection) GetClusterStatus() ([]ClusterStatusEntry, error) {
	var entries []ClusterStatusEntry
	if err := c.apiGet("/cluster/status", &entries); err != nil {
		return nil, fmt.Errorf("failed to get cluster status: %w", err)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Cluster options
// ---------------------------------------------------------------------------

func (c *PveConnection) GetClusterOptions() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.apiGet("/cluster/options", &result); err != nil {
		return nil, fmt.Errorf("failed to get cluster options: %w", err)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// HA resources
// ---------------------------------------------------------------------------

type HAResource struct {
	SID         string `json:"sid"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Node        string `json:"node"`
	MaxRestart  int    `json:"max_restart"`
	MaxRelocate int    `json:"max_relocate"`
	State       string `json:"state"`
	Group       string `json:"group"`
}

func (c *PveConnection) GetHAResources() ([]HAResource, error) {
	var resources []HAResource
	if err := c.apiGet("/cluster/ha/resources", &resources); err != nil {
		return nil, fmt.Errorf("failed to get HA resources: %w", err)
	}
	return resources, nil
}

// ---------------------------------------------------------------------------
// Cluster-level firewall rules
// ---------------------------------------------------------------------------

func (c *PveConnection) GetClusterFirewallRules() ([]FirewallRule, error) {
	var rules []FirewallRule
	if err := c.apiGet("/cluster/firewall/rules", &rules); err != nil {
		return nil, fmt.Errorf("failed to get cluster firewall rules: %w", err)
	}
	return rules, nil
}

// ---------------------------------------------------------------------------
// HA groups
// ---------------------------------------------------------------------------

type HAGroup struct {
	Group      string `json:"group"`
	Nodes      string `json:"nodes"` // comma-separated; members may have priorities like "pve1:2"
	Restricted int    `json:"restricted"`
	NoFailback int    `json:"nofailback"`
	Comment    string `json:"comment"`
	Type       string `json:"type"`
}

func (c *PveConnection) GetHAGroups() ([]HAGroup, error) {
	var groups []HAGroup
	if err := c.apiGet("/cluster/ha/groups", &groups); err != nil {
		return nil, fmt.Errorf("failed to get HA groups: %w", err)
	}
	return groups, nil
}
