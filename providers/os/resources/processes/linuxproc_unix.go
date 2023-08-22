// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package processes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

var UNIX_INODE_REGEX = regexp.MustCompile(`^socket:\[(\d+)\]$`)

// Read out all connected sockets
// we will ignore all FD errors here since we may not have access to everything
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
		fdInfo, err := lpm.conn.FileSystem().Stat(fdPath)
		if err != nil {
			continue
		}

		if fdInfo.Mode()&fs.ModeSocket == 0 {
			continue
		}

		inode, err := lpm.getInodeFromFd(fdPath)
		if err != nil {
			log.Error().Err(err).Msg("cannot get inode for fd")
			continue
		}

		res = append(res, inode)
	}

	return res, nil
}

func (lpm *LinuxProcManager) getInodeFromFd(fdPath string) (int64, error) {
	var inode int64
	command := fmt.Sprintf("readlink %s", fdPath)
	c, err := lpm.conn.RunCommand(command)
	if err != nil {
		return inode, fmt.Errorf("processes> could not run command: %v", err)
	}
	return readInodeFromOutput(c.Stdout)
}

func readInodeFromOutput(reader io.Reader) (int64, error) {
	var inode int64
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return inode, fmt.Errorf("processes> could not read command output: %v", err)
	}
	line := strings.TrimSuffix(buf.String(), "\n")
	if line == "" {
		return inode, fmt.Errorf("processes> could not get inode from fd")
	}
	m := UNIX_INODE_REGEX.FindStringSubmatch(line)
	inode, err = strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return inode, fmt.Errorf("processes> could not parse inode: %v", err)
	}
	return inode, nil
}
