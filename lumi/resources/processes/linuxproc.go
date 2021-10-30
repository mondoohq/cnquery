package processes

import (
	"io/fs"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	"go.mondoo.io/mondoo/motor"
)

type LinuxProcManager struct {
	motor *motor.Motor
}

func (lpm *LinuxProcManager) Name() string {
	return "Linux Process Manager"
}

func (lpm *LinuxProcManager) List() ([]*OSProcess, error) {
	// get all subdirectories of /proc, filter by nunbers
	f, err := lpm.motor.Transport.FS().Open("/proc")
	if err != nil {
		return nil, errors.WithMessage(err, "failed to access /proc")
	}

	dirs, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	res := []*OSProcess{}
	for i := range dirs {
		// we only parse directories that are numbers
		pid, err := strconv.ParseInt(dirs[i], 10, 64)
		if err != nil {
			continue
		}

		// collect process info
		proc, err := lpm.Process(pid)
		if err != nil {
			log.Warn().Err(err).Int64("pid", pid).Msg("lumi[processes]> could not retrieve process information")
			continue
		}

		res = append(res, proc)
	}
	return res, nil
}

// check that the pid directory exists
func (lpm *LinuxProcManager) Exists(pid int64) (bool, error) {
	trans := lpm.motor.Transport
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))
	afutil := afero.Afero{Fs: trans.FS()}
	return afutil.Exists(pidPath)
}

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

func (lpm *LinuxProcManager) Process(pid int64) (*OSProcess, error) {
	trans := lpm.motor.Transport
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))

	exists, err := lpm.Exists(pid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist: " + pidPath)
	}

	// parse the cmdline
	cmdlinef, err := trans.FS().Open(filepath.Join(pidPath, "cmdline"))
	if err != nil {
		return nil, err
	}
	defer cmdlinef.Close()

	cmdline, err := procfs.ParseProcessCmdline(cmdlinef)
	if err != nil {
		return nil, err
	}

	statusf, err := trans.FS().Open(filepath.Join(pidPath, "status"))
	if err != nil {
		return nil, err
	}
	defer statusf.Close()

	status, err := procfs.ParseProcessStatus(statusf)
	if err != nil {
		return nil, err
	}

	socketInodes := lpm.procSocketInods(pid, pidPath)

	process := &OSProcess{
		Pid:          pid,
		Executable:   status.Executable,
		State:        status.State,
		Command:      cmdline,
		SocketInodes: socketInodes,
	}

	return process, nil
}
