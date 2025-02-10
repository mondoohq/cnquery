// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

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
	t.Run("single device", func(t *testing.T) {
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
	})

	t.Run("multiple devices", func(t *testing.T) {
		devices := scsiDevices{
			{Lun: 0, VolumePath: "/dev/sda"},
			{Lun: 1, VolumePath: "/dev/sdb"},
			{Lun: 1, VolumePath: "/dev/sdc"},
			{Lun: 3, VolumePath: "/dev/sdd"},
		}

		filtered := filterScsiDevices(devices, 1)
		expected := scsiDevices{
			{Lun: 1, VolumePath: "/dev/sdb"},
			{Lun: 1, VolumePath: "/dev/sdc"},
		}
		assert.ElementsMatch(t, expected, filtered)

		filtered = filterScsiDevices(devices, 4)
		assert.Len(t, filtered, 0)
	})
}
