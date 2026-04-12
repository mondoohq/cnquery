// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

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
