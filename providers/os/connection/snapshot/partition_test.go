// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNoBootVolume(t *testing.T) {
	t.Run("is not boot", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "xfs",
			Label:      "label",
			Name:       "sda2",
			MountPoint: "",
		}
		require.True(t, block.isNoBootVolume())
	})

	t.Run("is boot (boot label)", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "vfat",
			Label:      "BOOT",
			Name:       "sda1",
			MountPoint: "/boot",
		}
		require.False(t, block.isNoBootVolume())
	})

	t.Run("is boot (vfat label)", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "vfat",
			Label:      "vfat",
			Name:       "sda1",
			MountPoint: "/boot",
		}
		require.False(t, block.isNoBootVolume())
	})

	t.Run("is boot (efi label)", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "vfat",
			Label:      "efi",
			Name:       "sda1",
			MountPoint: "/boot",
		}
		require.False(t, block.isNoBootVolume())
	})

	t.Run("is boot (empty uuid)", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "",
			FsType:     "vfat",
			Label:      "test",
			Name:       "sda1",
			MountPoint: "/boot",
		}
		require.False(t, block.isNoBootVolume())
	})
}

func TestIsMounted(t *testing.T) {
	t.Run("is mounted", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "xfs",
			Label:      "label",
			Name:       "sda2",
			MountPoint: "/mnt",
		}
		require.True(t, block.isMounted())
	})

	t.Run("is not mounted", func(t *testing.T) {
		block := BlockDevice{
			Uuid:       "12345",
			FsType:     "xfs",
			Label:      "label",
			Name:       "sda2",
			MountPoint: "",
		}
		require.False(t, block.isMounted())
	})
}
