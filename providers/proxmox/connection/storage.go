// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Storage pools
// ---------------------------------------------------------------------------

type StorageInfo struct {
	Storage  string  `json:"storage"`
	Type     string  `json:"type"`
	Content  string  `json:"content"`
	Path     string  `json:"path"`
	Enabled  int     `json:"enabled"`
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
	return storages, nil
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
