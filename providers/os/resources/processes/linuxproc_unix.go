// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package processes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// procSocketInods reads all connected sockets for a process by reading symlinks
// directly via os.Readlink instead of spawning a command per FD. This is safe
// because LinuxProcManager is only used for local connections where /proc is
// directly accessible.
func (lpm *LinuxProcManager) procSocketInods(pid int64, procPidPath string) ([]int64, error) {
	fdDirPath := filepath.Join(procPidPath, "fd")

	fdDir, err := lpm.conn.FileSystem().Open(fdDirPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil, fs.ErrPermission
		}
		return nil, err
	}

	fds, err := fdDir.Readdirnames(-1)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil, fs.ErrPermission
		}
		return nil, err
	}

	var res []int64
	for i := range fds {
		fdPath := filepath.Join(fdDirPath, fds[i])

		// Use os.Readlink (a single syscall) instead of Stat + RunCommand("readlink").
		// This avoids spawning a process per FD, which was the main bottleneck.
		link, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}

		// Only parse socket symlinks: "socket:[<inode>]"
		if !strings.HasPrefix(link, "socket:[") || !strings.HasSuffix(link, "]") {
			continue
		}

		inodeStr := link[len("socket:[") : len(link)-1]
		inode, err := strconv.ParseInt(inodeStr, 10, 64)
		if err != nil {
			continue
		}

		res = append(res, inode)
	}

	return res, nil
}
