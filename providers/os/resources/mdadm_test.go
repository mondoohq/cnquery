// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMdadmScan(t *testing.T) {
	input := `ARRAY /dev/md0 metadata=1.2 name=host:0 UUID=12345678:abcdef01:23456789:abcdef01
ARRAY /dev/md127 metadata=1.2 name=host:127 UUID=87654321:abcdef01:23456789:abcdef01
`
	names := parseMdadmScan(input)
	require.Len(t, names, 2)
	assert.Equal(t, "/dev/md0", names[0])
	assert.Equal(t, "/dev/md127", names[1])
}

func TestParseMdadmScanEmpty(t *testing.T) {
	names := parseMdadmScan("")
	assert.Empty(t, names)
}

func TestParseMdadmDetail(t *testing.T) {
	input := `/dev/md0:
           Version : 1.2
     Creation Time : Mon Jan  1 00:00:00 2024
        Raid Level : raid1
        Array Size : 1048576 (1024.00 MiB 1073.74 MB)
     Used Dev Size : 1048576 (1024.00 MiB 1073.74 MB)
      Raid Devices : 2
     Total Devices : 2
       Persistence : Superblock is persistent

             State : clean
    Active Devices : 2
   Working Devices : 2
    Failed Devices : 0
     Spare Devices : 0

              UUID : 12345678:abcdef01:23456789:abcdef01

    Number   Major   Minor   RaidDevice   State
       0       8        1        0      active sync   /dev/sda1
       1       8       17        1      active sync   /dev/sdb1
`
	arr := parseMdadmDetail(input)
	assert.Equal(t, "raid1", arr.level)
	assert.Equal(t, "clean", arr.state)
	assert.Equal(t, int64(2), arr.activeDevices)
	assert.Equal(t, int64(2), arr.workingDevices)
	assert.Equal(t, int64(0), arr.failedDevices)
	assert.Equal(t, int64(0), arr.spareDevices)
	assert.Equal(t, int64(1048576), arr.size)
	assert.Equal(t, "12345678:abcdef01:23456789:abcdef01", arr.uuid)
	assert.Equal(t, float64(-1), arr.resyncProgress)

	require.Len(t, arr.devices, 2)
	assert.Equal(t, "/dev/sda1", arr.devices[0].name)
	assert.Equal(t, int64(0), arr.devices[0].role)
	assert.Equal(t, "active sync", arr.devices[0].state)
	assert.Equal(t, "/dev/sdb1", arr.devices[1].name)
	assert.Equal(t, int64(1), arr.devices[1].role)
	assert.Equal(t, "active sync", arr.devices[1].state)
}

func TestParseMdadmDetailDegraded(t *testing.T) {
	input := `/dev/md0:
        Raid Level : raid5
        Array Size : 2097152 (2.00 GiB 2.15 GB)
             State : degraded
    Active Devices : 2
   Working Devices : 2
    Failed Devices : 1
     Spare Devices : 0
              UUID : aaaaaaaa:bbbbbbbb:cccccccc:dddddddd
    Rebuild Status : 45% complete

    Number   Major   Minor   RaidDevice   State
       0       8        1        0      active sync   /dev/sda1
       -       0        0        1      removed
       2       8       33        2      active sync   /dev/sdc1
`
	arr := parseMdadmDetail(input)
	assert.Equal(t, "raid5", arr.level)
	assert.Equal(t, "degraded", arr.state)
	assert.Equal(t, int64(1), arr.failedDevices)
	assert.Equal(t, float64(45), arr.resyncProgress)

	require.Len(t, arr.devices, 2)
	assert.Equal(t, "/dev/sda1", arr.devices[0].name)
	assert.Equal(t, "/dev/sdc1", arr.devices[1].name)
}

func TestParseMdadmDetailSpare(t *testing.T) {
	input := `/dev/md0:
        Raid Level : raid1
        Array Size : 1048576 (1024.00 MiB 1073.74 MB)
             State : clean
    Active Devices : 2
   Working Devices : 3
    Failed Devices : 0
     Spare Devices : 1
              UUID : 11111111:22222222:33333333:44444444

    Number   Major   Minor   RaidDevice   State
       0       8        1        0      active sync   /dev/sda1
       1       8       17        1      active sync   /dev/sdb1
       2       8       33       -1      spare   /dev/sdc1
`
	arr := parseMdadmDetail(input)
	assert.Equal(t, int64(1), arr.spareDevices)
	require.Len(t, arr.devices, 3)
	assert.Equal(t, "/dev/sdc1", arr.devices[2].name)
	assert.Equal(t, int64(-1), arr.devices[2].role)
	assert.Equal(t, "spare", arr.devices[2].state)
}

func TestParseRebuildPercent(t *testing.T) {
	assert.Equal(t, float64(45), parseRebuildPercent("45% complete"))
	assert.Equal(t, float64(99.9), parseRebuildPercent("99.9% complete"))
	assert.Equal(t, float64(-1), parseRebuildPercent(""))
	assert.Equal(t, float64(-1), parseRebuildPercent("unknown"))
}
