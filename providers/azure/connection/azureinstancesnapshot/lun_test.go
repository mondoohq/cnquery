// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLsscsiOutput(t *testing.T) {
	// different padding for the device names on purpose + an extra blank line
	output := `
	[0:0:0:0]    /dev/sda
	[0:0:1:1]     /dev/sdb
	[0:0:1:2]     /dev/sdc
	[0:0:0:3]   /dev/sdd
	
	`
	devices, err := parseLsscsiOutput(output)
	assert.NoError(t, err)
	assert.Len(t, devices, 4)
	expected := scsiDevices{
		{Lun: 0, VolumePath: "/dev/sda"},
		{Lun: 1, VolumePath: "/dev/sdb"},
		{Lun: 2, VolumePath: "/dev/sdc"},
		{Lun: 3, VolumePath: "/dev/sdd"},
	}
	assert.ElementsMatch(t, expected, devices)
}

func TestFilterScsiDevices(t *testing.T) {
	devices := scsiDevices{
		{Lun: 0, VolumePath: "/dev/sda"},
		{Lun: 1, VolumePath: "/dev/sdb"},
		{Lun: 2, VolumePath: "/dev/sdc"},
		{Lun: 3, VolumePath: "/dev/sdd"},
	}

	filtered := filterScsiDevices(devices, int32(1))
	expected := scsiDevices{
		{Lun: 1, VolumePath: "/dev/sdb"},
	}
	assert.ElementsMatch(t, expected, filtered)

	filtered = filterScsiDevices(devices, int32(4))
	assert.Len(t, filtered, 0)
}

func TestFindDeviceByBlock(t *testing.T) {
	devices := scsiDevices{
		{Lun: 0, VolumePath: "/dev/sda"},
		{Lun: 0, VolumePath: "/dev/sdb"},
	}
	t.Run("find device by block", func(t *testing.T) {
		blockDevices := &blockDevices{
			BlockDevices: []blockDevice{
				{
					Name: "sda",
					Children: []blockDevice{
						{
							Name:       "sda1",
							Mountpoint: []string{"/"},
						},
					},
				},
				{
					Name: "sdb",
					Children: []blockDevice{
						{
							Name:       "sdb1",
							Mountpoint: []string{""},
						},
					},
				},
			},
		}
		target, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.NoError(t, err)
		expected := "/dev/sdb"
		assert.Equal(t, expected, target)
	})

	t.Run("no matches", func(t *testing.T) {
		blockDevices := &blockDevices{
			BlockDevices: []blockDevice{
				{
					Name: "sdc",
					Children: []blockDevice{
						{
							Name:       "sdc1",
							Mountpoint: []string{"/"},
						},
					},
				},
				{
					Name: "sdc",
					Children: []blockDevice{
						{
							Name:       "sdc1",
							Mountpoint: []string{"/tmp"},
						},
					},
				},
			},
		}
		_, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.Error(t, err)
	})
	t.Run("empty target as all blocks are mounted", func(t *testing.T) {
		blockDevices := &blockDevices{
			BlockDevices: []blockDevice{
				{
					Name: "sda",
					Children: []blockDevice{
						{
							Name:       "sda1",
							Mountpoint: []string{"/"},
						},
					},
				},
				{
					Name: "sdb",
					Children: []blockDevice{
						{
							Name:       "sdb1",
							Mountpoint: []string{"/tmp"},
						},
					},
				},
			},
		}
		target, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.NoError(t, err)
		assert.Empty(t, target)
	})
}
