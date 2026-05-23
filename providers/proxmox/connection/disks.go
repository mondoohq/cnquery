// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Physical disks
// ---------------------------------------------------------------------------

// DiskInfo is one row from /nodes/<node>/disks/list.
type DiskInfo struct {
	DevPath  string `json:"devpath"`
	Size     int64  `json:"size"`
	Model    string `json:"model"`
	Vendor   string `json:"vendor"`
	Serial   string `json:"serial"`
	Used     string `json:"used"` // "ZFS", "LVM", "partitions", or ""
	ByIDLink string `json:"by_id_link"`
	WWN      string `json:"wwn"`
	RPM      int    `json:"rpm"`
	Type     string `json:"type"`   // hdd, ssd, nvme, usb
	Health   string `json:"health"` // PASSED, FAILED, UNKNOWN
	GPT      int    `json:"gpt"`
}

func (c *PveConnection) GetNodeDisks(node string) ([]DiskInfo, error) {
	var disks []DiskInfo
	path := fmt.Sprintf("/nodes/%s/disks/list", node)
	if err := c.apiGet(path, &disks); err != nil {
		return nil, fmt.Errorf("failed to list disks on node %s: %w", node, err)
	}
	return disks, nil
}

// DiskSMART is the body returned by /nodes/<node>/disks/smart?disk=<dev>.
// `Type` is the SMART report flavor ("ata", "nvme", "sas", "text") and
// `Health` mirrors the overall PASSED/FAILED status the device reports.
type DiskSMART struct {
	Health     string           `json:"health"`
	Type       string           `json:"type"`
	Attributes []SMARTAttribute `json:"attributes"`
	// `Text` is populated when Proxmox can only get unstructured SMART
	// output from the device (e.g. some NVMe drives).
	Text string `json:"text"`
}

type SMARTAttribute struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Value     int    `json:"value"`
	Worst     int    `json:"worst"`
	Threshold int    `json:"threshold"`
	Raw       string `json:"raw"`
	Flags     string `json:"flags"`
	Fail      string `json:"fail"`
}

func (c *PveConnection) GetDiskSMART(node, devpath string) (*DiskSMART, error) {
	var smart DiskSMART
	path := fmt.Sprintf("/nodes/%s/disks/smart?disk=%s", node, devpath)
	if err := c.apiGet(path, &smart); err != nil {
		return nil, fmt.Errorf("failed to get SMART for %s on node %s: %w", devpath, node, err)
	}
	return &smart, nil
}

// ---------------------------------------------------------------------------
// ZFS pools
// ---------------------------------------------------------------------------

type ZFSPoolInfo struct {
	Name   string  `json:"name"`
	Size   int64   `json:"size"`
	Alloc  int64   `json:"alloc"`
	Free   int64   `json:"free"`
	Frag   int     `json:"frag"`
	Dedup  float64 `json:"dedup"`
	Health string  `json:"health"`
}

func (c *PveConnection) GetNodeZFSPools(node string) ([]ZFSPoolInfo, error) {
	var pools []ZFSPoolInfo
	path := fmt.Sprintf("/nodes/%s/disks/zfs", node)
	if err := c.apiGet(path, &pools); err != nil {
		return nil, fmt.Errorf("failed to list ZFS pools on node %s: %w", node, err)
	}
	return pools, nil
}

// ZFSPoolDetail is the per-pool response with children (vdevs).
type ZFSPoolDetail struct {
	Name     string         `json:"name"`
	State    string         `json:"state"`
	Scan     string         `json:"scan"`
	Errors   string         `json:"errors"`
	Children []ZFSPoolChild `json:"children"`
}

type ZFSPoolChild struct {
	Name     string         `json:"name"`
	State    string         `json:"state"`
	Read     int            `json:"read"`
	Write    int            `json:"write"`
	Cksum    int            `json:"cksum"`
	Msg      string         `json:"msg"`
	Type     string         `json:"type"`
	Leaf     int            `json:"leaf"`
	Children []ZFSPoolChild `json:"children"`
}

func (c *PveConnection) GetZFSPoolDetail(node, pool string) (*ZFSPoolDetail, error) {
	var detail ZFSPoolDetail
	path := fmt.Sprintf("/nodes/%s/disks/zfs/%s", node, pool)
	if err := c.apiGet(path, &detail); err != nil {
		return nil, fmt.Errorf("failed to get ZFS pool %s on node %s: %w", pool, node, err)
	}
	return &detail, nil
}

// ---------------------------------------------------------------------------
// LVM volume groups + thin pools
// ---------------------------------------------------------------------------

type LVMTreeResponse struct {
	Children []LVMVolumeGroup `json:"children"`
}

type LVMVolumeGroup struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Free    int64  `json:"free"`
	LVCount int    `json:"lvcount"`
	Leaf    int    `json:"leaf"`
}

func (c *PveConnection) GetNodeLVM(node string) ([]LVMVolumeGroup, error) {
	var resp LVMTreeResponse
	path := fmt.Sprintf("/nodes/%s/disks/lvm", node)
	if err := c.apiGet(path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LVM tree on node %s: %w", node, err)
	}
	return resp.Children, nil
}

type LVMThinPool struct {
	LV           string `json:"lv"`
	VG           string `json:"vg"`
	LVSize       int64  `json:"lv_size"`
	UsedSize     int64  `json:"used"`
	MetadataSize int64  `json:"metadata_size"`
	MetadataUsed int64  `json:"metadata_used"`
}

func (c *PveConnection) GetNodeLVMThin(node string) ([]LVMThinPool, error) {
	var pools []LVMThinPool
	path := fmt.Sprintf("/nodes/%s/disks/lvmthin", node)
	if err := c.apiGet(path, &pools); err != nil {
		return nil, fmt.Errorf("failed to get LVM-thin pools on node %s: %w", node, err)
	}
	return pools, nil
}
