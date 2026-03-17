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

	"github.com/spf13/afero"
)

// ParseSocketInode extracts the inode number from a /proc/*/fd/* symlink target.
// Returns -1 if the link target is not a socket (e.g. "pipe:[...]", "/dev/null").
func ParseSocketInode(link string) (int64, error) {
	if !strings.HasPrefix(link, "socket:[") || !strings.HasSuffix(link, "]") {
		return -1, nil
	}
	inodeStr := link[len("socket:[") : len(link)-1]
	return strconv.ParseInt(inodeStr, 10, 64)
}

// procSocketInods reads all connected sockets for a process using the
// connection's filesystem abstraction. It uses afero.LinkReader (ReadlinkIfPossible)
// to resolve symlinks instead of spawning a command per FD.
func (lpm *LinuxProcManager) procSocketInods(_ int64, procPidPath string) ([]int64, error) {
	connFs := lpm.conn.FileSystem()
	fdDirPath := filepath.Join(procPidPath, "fd")

	fdDir, err := connFs.Open(fdDirPath)
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

	lr, ok := connFs.(afero.LinkReader)
	if !ok {
		return nil, errors.New("filesystem does not support readlink")
	}

	var res []int64
	for i := range fds {
		fdPath := filepath.Join(fdDirPath, fds[i])

		link, err := lr.ReadlinkIfPossible(fdPath)
		if err != nil {
			continue
		}

		inode, err := ParseSocketInode(link)
		if err != nil || inode < 0 {
			continue
		}

		res = append(res, inode)
	}

	return res, nil
}
