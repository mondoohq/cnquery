// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "testing"

// TestNormalizeClusterStorageEnabled verifies Enabled is derived from the
// cluster "disable" config key. The cluster /storage endpoint omits the runtime
// "enabled" key, so without normalization every cluster storage read as
// enabled=false.
func TestNormalizeClusterStorageEnabled(t *testing.T) {
	storages := []StorageInfo{
		{Storage: "cluster-enabled", Disable: 0, Enabled: 0},  // no disable key -> enabled
		{Storage: "cluster-disabled", Disable: 1, Enabled: 0}, // disable=1 -> disabled
		{Storage: "node-enabled", Disable: 0, Enabled: 1},     // explicit enabled preserved
		{Storage: "disable-wins", Disable: 1, Enabled: 1},     // disable overrides explicit enabled
	}

	normalizeClusterStorageEnabled(storages)

	want := []int{1, 0, 1, 0}
	for i, s := range storages {
		if s.Enabled != want[i] {
			t.Errorf("%s: Enabled = %d, want %d", s.Storage, s.Enabled, want[i])
		}
	}
}
