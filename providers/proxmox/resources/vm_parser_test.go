// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"go.mondoo.com/mql/v13/llx"
)

func getStr(m map[string]*llx.RawData, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.Value.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]*llx.RawData, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.Value.(bool); ok {
			return b
		}
	}
	return false
}

func getInt(m map[string]*llx.RawData, key string) int64 {
	if v, ok := m[key]; ok {
		if i, ok := v.Value.(int64); ok {
			return i
		}
	}
	return 0
}

func TestParseVMNetworkConfig_Virtio(t *testing.T) {
	net := parseVMNetworkConfig("net0", "virtio=BC:24:11:AA:BB:CC,bridge=vmbr0,firewall=1,tag=100")

	if v := getStr(net, "model"); v != "virtio" {
		t.Errorf("expected model 'virtio', got %q", v)
	}
	if v := getStr(net, "macAddress"); v != "BC:24:11:AA:BB:CC" {
		t.Errorf("expected MAC 'BC:24:11:AA:BB:CC', got %q", v)
	}
	if v := getStr(net, "bridge"); v != "vmbr0" {
		t.Errorf("expected bridge 'vmbr0', got %q", v)
	}
	if !getBool(net, "firewall") {
		t.Error("expected firewall=true")
	}
	if v := getInt(net, "tag"); v != 100 {
		t.Errorf("expected tag 100, got %d", v)
	}
}

func TestParseVMNetworkConfig_E1000(t *testing.T) {
	net := parseVMNetworkConfig("net1", "e1000=AA:BB:CC:DD:EE:FF,bridge=vmbr1")

	if v := getStr(net, "model"); v != "e1000" {
		t.Errorf("expected model 'e1000', got %q", v)
	}
	if v := getStr(net, "bridge"); v != "vmbr1" {
		t.Errorf("expected bridge 'vmbr1', got %q", v)
	}
	if getBool(net, "firewall") {
		t.Error("expected firewall=false")
	}
}

func TestParseVMDiskConfig_Full(t *testing.T) {
	disk := parseVMDiskConfig("scsi0", "local-lvm:vm-100-disk-0,size=32G,cache=writeback,iothread=1,backup=0")

	if v := getStr(disk, "storage"); v != "local-lvm" {
		t.Errorf("expected storage 'local-lvm', got %q", v)
	}
	if v := getInt(disk, "size"); v != 32*1024*1024*1024 {
		t.Errorf("expected size 32GiB, got %d", v)
	}
	if v := getStr(disk, "cache"); v != "writeback" {
		t.Errorf("expected cache 'writeback', got %q", v)
	}
	if !getBool(disk, "iothread") {
		t.Error("expected iothread=true")
	}
	if getBool(disk, "backup") {
		t.Error("expected backup=false")
	}
}

func TestParseVMDiskConfig_Minimal(t *testing.T) {
	disk := parseVMDiskConfig("virtio0", "ceph-pool:vm-200-disk-0,size=100G")

	if v := getStr(disk, "storage"); v != "ceph-pool" {
		t.Errorf("expected storage 'ceph-pool', got %q", v)
	}
	if v := getInt(disk, "size"); v != 100*1024*1024*1024 {
		t.Errorf("expected size 100GiB, got %d", v)
	}
	if !getBool(disk, "backup") {
		t.Error("expected backup=true (default)")
	}
}

func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"32G", 32 * 1024 * 1024 * 1024},
		{"512M", 512 * 1024 * 1024},
		{"1T", 1024 * 1024 * 1024 * 1024},
		{"4096K", 4096 * 1024},
		{"10", 10},
	}
	for _, tt := range tests {
		result := parseSizeToBytes(tt.input)
		if result != tt.expected {
			t.Errorf("parseSizeToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestLooksLikeMAC(t *testing.T) {
	if !looksLikeMAC("AA:BB:CC:DD:EE:FF") {
		t.Error("expected true for valid MAC")
	}
	if looksLikeMAC("not-a-mac") {
		t.Error("expected false for non-MAC")
	}
	if looksLikeMAC("") {
		t.Error("expected false for empty string")
	}
}
