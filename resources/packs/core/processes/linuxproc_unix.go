//go:build !windows

package processes

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

// Read out all connected sockets
// we will ignore all FD errors here since we may not have access to everything
func (lpm *LinuxProcManager) procSocketInods(pid int64, procPidPath string) ([]int64, error) {
	fdDirPath := filepath.Join(procPidPath, "fd")

	fdDir, err := lpm.provider.FS().Open(fdDirPath)
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
		fdInfo, err := lpm.provider.FS().Stat(fdPath)
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
	c, err := lpm.provider.RunCommand(command)
	if err != nil {
		return inode, fmt.Errorf("processes> could not run command: %v", err)
	}
	scannerInode := bufio.NewScanner(c.Stdout)
	scannerInode.Scan()
	m := UNIX_INODE_REGEX.FindStringSubmatch(scannerInode.Text())
	inode, err = strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return inode, fmt.Errorf("processes> could not parse inode: %v", err)
	}
	return inode, nil
}
