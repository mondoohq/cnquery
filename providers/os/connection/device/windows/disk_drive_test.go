// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterDrives(t *testing.T) {
	t.Run("filter by serial number", func(t *testing.T) {
		opts := map[string]string{
			SerialNumberOption: "1234",
		}
		drives := []*diskDrive{
			{
				SerialNumber:    "1234",
				SCSILogicalUnit: 0,
				Index:           0,
				Name:            "a",
			},
			{
				SerialNumber:    "5678",
				SCSILogicalUnit: 1,
				Index:           1,
				Name:            "b",
			},
		}
		filtered, err := filterDiskDrives(drives, opts)
		require.NoError(t, err)
		expected := &diskDrive{
			SerialNumber:    "1234",
			SCSILogicalUnit: 0,
			Index:           0,
			Name:            "a",
		}
		require.Equal(t, expected, filtered)
	})

	t.Run("filter by LUN", func(t *testing.T) {
		opts := map[string]string{
			LunOption: "1",
		}
		drives := []*diskDrive{
			{
				SerialNumber:    "1234",
				SCSILogicalUnit: 0,
				Index:           0,
				Name:            "a",
			},
			{
				SerialNumber:    "5678",
				SCSILogicalUnit: 1,
				Index:           1,
				Name:            "b",
			},
		}
		filtered, err := filterDiskDrives(drives, opts)
		require.NoError(t, err)
		expected := &diskDrive{
			SerialNumber:    "5678",
			SCSILogicalUnit: 1,
			Index:           1,
			Name:            "b",
		}
		require.Equal(t, expected, filtered)
	})
	t.Run("filter by invalid LUN", func(t *testing.T) {
		opts := map[string]string{
			LunOption: "a",
		}
		drives := []*diskDrive{
			{
				SerialNumber:    "1234",
				SCSILogicalUnit: 0,
				Index:           0,
				Name:            "a",
			},
			{
				SerialNumber:    "5678",
				SCSILogicalUnit: 1,
				Index:           1,
				Name:            "b",
			},
		}
		_, err := filterDiskDrives(drives, opts)
		require.Error(t, err)
	})
}

func TestFilterPartitions(t *testing.T) {
	t.Run("find a partition (Basic type)", func(t *testing.T) {
		parts := []*diskPartition{
			{
				DriveLetter: "A",
				Size:        123,
				Type:        "Basic",
			},
			{
				Size: 123,
				Type: "Basic",
			},
		}
		part, err := filterPartitions(parts)
		require.NoError(t, err)
		expected := &diskPartition{
			DriveLetter: "A",
			Size:        123,
			Type:        "Basic",
		}
		require.Equal(t, expected, part)
	})

	t.Run("find a partition (IFS type)", func(t *testing.T) {
		parts := []*diskPartition{
			{
				DriveLetter: "A",
				Size:        123,
				Type:        "IFS",
			},
			{
				Size: 123,
				Type: "Basic",
			},
		}
		part, err := filterPartitions(parts)
		require.NoError(t, err)
		expected := &diskPartition{
			DriveLetter: "A",
			Size:        123,
			Type:        "IFS",
		}
		require.Equal(t, expected, part)
	})

	t.Run("no applicable partition", func(t *testing.T) {
		parts := []*diskPartition{
			{
				Size: 123,
				Type: "IFS",
			},
			{
				Size: 123,
				Type: "Basic",
			},
		}
		_, err := filterPartitions(parts)
		require.Error(t, err)
	})
}
