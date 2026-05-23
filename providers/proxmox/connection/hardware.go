// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"net/url"
)

// ---------------------------------------------------------------------------
// Node PCI devices
// ---------------------------------------------------------------------------

// NodePCIDevice mirrors /nodes/<node>/hardware/pci. The id is the
// canonical `<seg>:<bus>:<slot>.<func>` address (Proxmox emits both
// short and full forms depending on segment 0; we keep whatever the
// API hands back).
type NodePCIDevice struct {
	ID         string `json:"id"`
	Class      string `json:"class"`
	ClassName  string `json:"class_name"`
	Vendor     string `json:"vendor"`
	VendorName string `json:"vendor_name"`
	Device     string `json:"device"`
	DeviceName string `json:"device_name"`
	Subsystem  struct {
		Vendor     string `json:"vendor"`
		VendorName string `json:"vendor_name"`
		Device     string `json:"device"`
		DeviceName string `json:"device_name"`
	} `json:"subsystem"`
	IOMMUGroup int `json:"iommugroup"`
	MdevSupp   int `json:"mdev"`
}

func (c *PveConnection) GetNodePCIDevices(node string) ([]NodePCIDevice, error) {
	var devs []NodePCIDevice
	path := fmt.Sprintf("/nodes/%s/hardware/pci", url.PathEscape(node))
	if err := c.apiGet(path, &devs); err != nil {
		return nil, fmt.Errorf("failed to list PCI devices on node %s: %w", node, err)
	}
	return devs, nil
}

// ---------------------------------------------------------------------------
// Node USB devices
// ---------------------------------------------------------------------------

// NodeUSBDevice mirrors /nodes/<node>/hardware/usb. The shape uses
// hexadecimal vendor:product ids and decimal bus/port indices.
type NodeUSBDevice struct {
	BusNum       string `json:"busnum"`
	DevNum       string `json:"devnum"`
	Port         string `json:"port"`
	Level        string `json:"level"`
	Class        string `json:"class"`
	VendID       string `json:"vendid"`
	ProdID       string `json:"prodid"`
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	Serial       string `json:"serial"`
	Speed        string `json:"speed"`
	UsbPath      string `json:"usbpath"`
}

func (c *PveConnection) GetNodeUSBDevices(node string) ([]NodeUSBDevice, error) {
	var devs []NodeUSBDevice
	path := fmt.Sprintf("/nodes/%s/hardware/usb", url.PathEscape(node))
	if err := c.apiGet(path, &devs); err != nil {
		return nil, fmt.Errorf("failed to list USB devices on node %s: %w", node, err)
	}
	return devs, nil
}

// ---------------------------------------------------------------------------
// Cluster firewall security groups
// ---------------------------------------------------------------------------

type FirewallGroupInfo struct {
	Group   string `json:"group"`
	Comment string `json:"comment"`
	Digest  string `json:"digest"`
}

func (c *PveConnection) GetClusterFirewallGroups() ([]FirewallGroupInfo, error) {
	var groups []FirewallGroupInfo
	if err := c.apiGet("/cluster/firewall/groups", &groups); err != nil {
		return nil, fmt.Errorf("failed to get firewall security groups: %w", err)
	}
	return groups, nil
}

func (c *PveConnection) GetClusterFirewallGroupRules(group string) ([]FirewallRule, error) {
	var rules []FirewallRule
	path := fmt.Sprintf("/cluster/firewall/groups/%s", url.PathEscape(group))
	if err := c.apiGet(path, &rules); err != nil {
		return nil, fmt.Errorf("failed to get rules for firewall group %s: %w", group, err)
	}
	return rules, nil
}
