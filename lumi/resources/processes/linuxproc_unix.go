//go:build !windows

package processes

import (
	"io/fs"
	"path/filepath"
	"syscall"
)

// Read out all connected sockets
// we will ignore all FD errors here since we may not have access to everything
func (lpm *LinuxProcManager) procSocketInods(pid int64, procPidPath string) []int64 {
	trans := lpm.motor.Transport
	fdDirPath := filepath.Join(procPidPath, "fd")

	fdDir, err := lpm.motor.Transport.FS().Open(fdDirPath)
	if err != nil {
		return nil
	}

	fds, err := fdDir.Readdirnames(-1)
	if err != nil {
		return nil
	}

	var res []int64
	for i := range fds {
		fdPath := filepath.Join(fdDirPath, fds[i])
		fdInfo, err := trans.FS().Stat(fdPath)
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

	return res
}
