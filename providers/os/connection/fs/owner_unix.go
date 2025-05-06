// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package fs

import (
	"os"
	"syscall"
)

func (c *FileSystemConnection) fileowner(stat os.FileInfo) (float64, float64) {
	uid := float64(-1)
	gid := float64(-1)
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = float64(stat.Uid)
		gid = float64(stat.Gid)
	}
	return uid, gid
}
