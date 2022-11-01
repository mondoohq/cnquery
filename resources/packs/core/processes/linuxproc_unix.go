//go:build !windows

package processes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
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

		// At this point we need to get access to the inode of the file.
		// This is unfortunately not available via the fs.FileInfo field.
		// The inode is necessary to connect sockets of a process with running
		// ports that we find on the system.
		// TODO: needs fixing to work with remote connections via SSH
		raw := fdInfo.Sys()
		stat_t, ok := raw.(*syscall.Stat_t)
		if !ok {
			continue
		}

		res = append(res, int64(stat_t.Ino))
	}

	return res, nil
}
