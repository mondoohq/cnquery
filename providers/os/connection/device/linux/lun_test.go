// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLsscsiCLIOutput(t *testing.T) {
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

func TestParseScsiPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected scsiDeviceInfo
		err      bool
	}{
		{
			name: "valid path",
			path: "/sys/devices/LNXSYSTM:00/LNXSYBUS:00/ACPI0004:00/VMBUS:00/f8b3781a-1e82-4818-a1c3-63d806ec15bb/host0/target0:0:0/0:0:0:0/block/sda",
			expected: scsiDeviceInfo{
				Lun:        0,
				VolumePath: "/dev/sda",
			},
		},
		{
			name: "invalid path (short)",
			path: "/sys/devices/no-op",
			err:  true,
		},
		{
			name: "invalid path (not a block)",
			path: "/sys/devices/LNXSYSTM:00/LNXSYBUS:00/ACPI0004:00/VMBUS:00/f8b3781a-1e82-4818-a1c3-63d806ec15bb/host0/target0:0:0/0:0:0:0",
			err:  true,
		},
		{
			name:     "invalid path (invalid H:B:T:L)",
			path:     "/sys/devices/virtual/block/loop0",
			err:      true,
			expected: scsiDeviceInfo{VolumePath: "/dev/loop0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parseScsiDevicePath(tt.path)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, res)
		})
	}
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
