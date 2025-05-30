// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes

import (
	"path/filepath"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/procfs"
)

type LinuxProcManager struct {
	conn shared.Connection
}

func (lpm *LinuxProcManager) Name() string {
	return "Linux Process Manager"
}

func (lpm *LinuxProcManager) List() ([]*OSProcess, error) {
	// get all subdirectories of /proc, filter by numbers
	f, err := lpm.conn.FileSystem().Open("/proc")
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
			log.Warn().Err(err).Int64("pid", pid).Msg("mql[processes]> could not retrieve process information")
			continue
		}

		res = append(res, proc)
	}
	return res, nil
}

// check that the pid directory exists
func (lpm *LinuxProcManager) Exists(pid int64) (bool, error) {
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))
	afutil := afero.Afero{Fs: lpm.conn.FileSystem()}
	return afutil.Exists(pidPath)
}

func (lpm *LinuxProcManager) Process(pid int64) (*OSProcess, error) {
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))

	exists, err := lpm.Exists(pid)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist: " + pidPath)
	}

	// parse the cmdline
	cmdlinef, err := lpm.conn.FileSystem().Open(filepath.Join(pidPath, "cmdline"))
	if err != nil {
		return nil, err
	}
	defer cmdlinef.Close()

	cmdline, err := procfs.ParseProcessCmdline(cmdlinef)
	if err != nil {
		return nil, err
	}

	// If cmdline is empty (likely a kernel thread), read from /proc/<pid>/comm instead
	if cmdline == "" {
		commf, err := lpm.conn.FileSystem().Open(filepath.Join(pidPath, "comm"))
		if err != nil {
			return nil, err
		}
		defer commf.Close()

		cmdline, err = procfs.ParseProcessCmdline(commf)
		if err != nil {
			return nil, err
		}
	}

	statusf, err := lpm.conn.FileSystem().Open(filepath.Join(pidPath, "status"))
	if err != nil {
		return nil, err
	}
	defer statusf.Close()

	status, err := procfs.ParseProcessStatus(statusf)
	if err != nil {
		return nil, err
	}

	socketInodes, socketInodesErr := lpm.procSocketInods(pid, pidPath)

	process := &OSProcess{
		Pid:               pid,
		Executable:        status.Executable,
		State:             status.State,
		Command:           cmdline,
		SocketInodes:      socketInodes,
		SocketInodesError: socketInodesErr,
	}

	return process, nil
}

func (lpm *LinuxProcManager) ListSocketInodesByProcess() (map[int64]plugin.TValue[[]int64], error) {
	return nil, nil
}
