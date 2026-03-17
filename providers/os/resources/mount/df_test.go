// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mount

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDfLinux(t *testing.T) {
	input := `Filesystem     1024-blocks      Used Available Capacity Mounted on
udev              4017564         0   4017564       0% /dev
tmpfs              809396      1568    807828       1% /run
/dev/sda1        51340776  12165612  36537668      25% /
tmpfs             4046976         0   4046976       0% /dev/shm
/dev/sdb1       102400000  51200000  51200000      50% /mnt/data`

	entries := ParseDf(strings.NewReader(input))

	require.Len(t, entries, 5)

	root := entries["/"]
	require.NotNil(t, root)
	assert.Equal(t, "/dev/sda1", root.Filesystem)
	assert.Equal(t, int64(51340776*1024), root.Size)
	assert.Equal(t, int64(12165612*1024), root.Used)
	assert.Equal(t, int64(36537668*1024), root.Available)

	dev := entries["/dev"]
	require.NotNil(t, dev)
	assert.Equal(t, "udev", dev.Filesystem)
	assert.Equal(t, int64(0), dev.Used)
}

func TestParseDfMacOS(t *testing.T) {
	input := `Filesystem     1024-blocks      Used Available Capacity  Mounted on
/dev/disk3s1s1   971350180  12165612 492963572     3%    /
devfs                  218       218         0   100%    /dev
/dev/disk3s6     971350180    131076 492963572     1%    /System/Volumes/VM`

	entries := ParseDf(strings.NewReader(input))

	require.Len(t, entries, 3)

	root := entries["/"]
	require.NotNil(t, root)
	assert.Equal(t, "/dev/disk3s1s1", root.Filesystem)
	assert.Equal(t, int64(971350180*1024), root.Size)

	vm := entries["/System/Volumes/VM"]
	require.NotNil(t, vm)
	assert.Equal(t, "/dev/disk3s6", vm.Filesystem)
}

func TestParseDfEmpty(t *testing.T) {
	entries := ParseDf(strings.NewReader(""))
	assert.Empty(t, entries)
}

func TestParseDfHeaderOnly(t *testing.T) {
	input := `Filesystem     1024-blocks      Used Available Capacity Mounted on`
	entries := ParseDf(strings.NewReader(input))
	assert.Empty(t, entries)
}
