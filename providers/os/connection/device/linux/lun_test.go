// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
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

	filtered := filterScsiDevices(devices, 1)
	expected := scsiDevices{
		{Lun: 1, VolumePath: "/dev/sdb"},
	}
	assert.ElementsMatch(t, expected, filtered)

	filtered = filterScsiDevices(devices, 4)
	assert.Len(t, filtered, 0)
}

func TestFindDeviceByBlock(t *testing.T) {
	devices := scsiDevices{
		{Lun: 0, VolumePath: "/dev/sda"},
		{Lun: 0, VolumePath: "/dev/sdb"},
	}
	t.Run("find device by block", func(t *testing.T) {
		blockDevices := &snapshot.BlockDevices{
			BlockDevices: []snapshot.BlockDevice{
				{
					Name: "sda",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sda1",
							MountPoint: "/",
						},
					},
				},
				{
					Name: "sdb",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sdb1",
							MountPoint: "",
						},
					},
				},
			},
		}
		target, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.NoError(t, err)
		expected := blockDevices.BlockDevices[1]
		assert.Equal(t, expected, target)
	})

	t.Run("no matches", func(t *testing.T) {
		blockDevices := &snapshot.BlockDevices{
			BlockDevices: []snapshot.BlockDevice{
				{
					Name: "sdc",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sdc1",
							MountPoint: "/",
						},
					},
				},
				{
					Name: "sdc",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sdc1",
							MountPoint: "/tmp",
						},
					},
				},
			},
		}
		_, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.Error(t, err)
	})
	t.Run("empty target as all blocks are mounted", func(t *testing.T) {
		blockDevices := &snapshot.BlockDevices{
			BlockDevices: []snapshot.BlockDevice{
				{
					Name: "sda",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sda1",
							MountPoint: "/",
						},
					},
				},
				{
					Name: "sdb",
					Children: []snapshot.BlockDevice{
						{
							Name:       "sdb1",
							MountPoint: "/tmp",
						},
					},
				},
			},
		}
		_, err := findMatchingDeviceByBlock(devices, blockDevices)
		assert.Error(t, err)
	})
}
