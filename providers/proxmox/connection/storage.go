// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Storage pools
// ---------------------------------------------------------------------------

type StorageInfo struct {
	Storage string `json:"storage"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Path    string `json:"path"`
	Enabled int    `json:"enabled"`
	// Disable is the cluster /storage config key (1 = disabled); /nodes/<n>/storage
	// uses "enabled" instead. GetStorages normalizes Enabled from Disable.
	Disable  int     `json:"disable"`
	Shared   int     `json:"shared"`
	Total    int64   `json:"total"`
	Used     int64   `json:"used"`
	Avail    int64   `json:"avail"`
	UsedFrac float64 `json:"used_fraction"`
	// EncryptionKey carries the PBS encryption-key field. The value is
	// either an explicit key fingerprint or the literal "autogen" when
	// Proxmox manages the key. An empty string means encryption is off.
	EncryptionKey string `json:"encryption-key"`
}

func (c *PveConnection) GetStorages() ([]StorageInfo, error) {
	var storages []StorageInfo
	if err := c.apiGet("/storage", &storages); err != nil {
		return nil, fmt.Errorf("failed to get storages: %w", err)
	}
	normalizeClusterStorageEnabled(storages)
	return storages, nil
}

// normalizeClusterStorageEnabled derives Enabled from the cluster /storage
// "disable" config key. That endpoint reports config (disable), not the runtime
// "enabled" key that /nodes/<n>/storage returns, so without this every
// cluster-level storage reads as enabled=false. Enabled is only inferred when
// the response didn't already provide it, so an endpoint returning both keys
// keeps its explicit value.
func normalizeClusterStorageEnabled(storages []StorageInfo) {
	for i := range storages {
		if storages[i].Disable != 0 {
			storages[i].Enabled = 0
		} else if storages[i].Enabled == 0 {
			storages[i].Enabled = 1
		}
	}
}

// ---------------------------------------------------------------------------
// Resource pools
// ---------------------------------------------------------------------------

type PoolInfo struct {
	PoolID  string `json:"poolid"`
	Comment string `json:"comment"`
}

func (c *PveConnection) GetPools() ([]PoolInfo, error) {
	var pools []PoolInfo
	if err := c.apiGet("/pools", &pools); err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}
	return pools, nil
}
