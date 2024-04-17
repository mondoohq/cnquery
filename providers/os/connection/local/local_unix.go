// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package local

import (
	"os"
	"syscall"

	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

func (c *LocalConnection) fileowner(stat os.FileInfo) (int64, int64) {
	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*shared.FileInfo); ok {
		uid = stat.Uid
		gid = stat.Gid
	}
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	return uid, gid
}
