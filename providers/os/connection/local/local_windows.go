// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package local

import "os"

func (c *LocalConnection) fileowner(stat os.FileInfo) (int64, int64) {
	uid := int64(-1)
	gid := int64(-1)

	return uid, gid
}
