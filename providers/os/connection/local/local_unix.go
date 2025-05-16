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

func (c *LocalConnection) fileowner(stat os.FileInfo) (float64, float64) {
	uid := float64(-1)
	gid := float64(-1)
	if stat, ok := stat.Sys().(*shared.FileInfo); ok {
		uid = float64(stat.Uid)
		gid = float64(stat.Gid)
	}
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = float64(stat.Uid)
		gid = float64(stat.Gid)
	}
	return uid, gid
}
