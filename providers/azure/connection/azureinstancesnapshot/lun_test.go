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
	expected := []deviceInfo{
		{Lun: 0, VolumePath: "/dev/sda"},
		{Lun: 1, VolumePath: "/dev/sdb"},
		{Lun: 2, VolumePath: "/dev/sdc"},
		{Lun: 3, VolumePath: "/dev/sdd"},
	}
	assert.ElementsMatch(t, expected, devices)
}
