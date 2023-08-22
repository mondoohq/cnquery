// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package processes

import (
	"errors"
)

// Read out all connected sockets. This is not yet implemented on non-Unix
// systems and needs some work to function via remote connections
func (lpm *LinuxProcManager) procSocketInods(pid int64, procPidPath string) ([]int64, error) {
	return []int64{}, errors.New("reading socket inodes is not implemented for Windows")
}
