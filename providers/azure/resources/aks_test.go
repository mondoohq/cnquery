// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	clusters "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
	"github.com/stretchr/testify/assert"
)

func TestAksDiskCsiDriverEnabled(t *testing.T) {
	t.Run("nil storage profile returns nil", func(t *testing.T) {
		assert.Nil(t, aksDiskCsiDriverEnabled(nil))
	})
	t.Run("nil disk driver returns nil", func(t *testing.T) {
		assert.Nil(t, aksDiskCsiDriverEnabled(&clusters.ManagedClusterStorageProfile{}))
	})
	t.Run("enabled true", func(t *testing.T) {
		enabled := true
		sp := &clusters.ManagedClusterStorageProfile{
			DiskCSIDriver: &clusters.ManagedClusterStorageProfileDiskCSIDriver{Enabled: &enabled},
		}
		got := aksDiskCsiDriverEnabled(sp)
		assert.NotNil(t, got)
		assert.True(t, *got)
	})
}
