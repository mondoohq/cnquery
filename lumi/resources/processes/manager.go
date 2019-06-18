package processes

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

func ResolveManager(motor *motor.Motor) (OSProcessManager, error) {
	var pm OSProcessManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			pm = &LinuxProcessManager{motor: motor}
			break
		} else if platform.Family[i] == "unix" {
			pm = &UnixProcessManager{motor: motor}
			break
		}
	}
	return pm, nil
}

type OSProcess struct {
	Pid        int64
	Command    string
	Executable string
	State      string
	Uid        int64
}

type OSProcessManager interface {
	Name() string
	Exists(pid int64) (bool, error)
	Process(pid int64) (*OSProcess, error)
	List() ([]*OSProcess, error)
}

type LinuxProcessManager struct {
	motor *motor.Motor
}

func (lpm *LinuxProcessManager) Name() string {
	return "Linux Process Manager"
}

func (lpm *LinuxProcessManager) List() ([]*OSProcess, error) {
	// get all subdirectories of /proc, filter by nunbers
	f, err := lpm.motor.Transport.File("/proc")
	if err != nil {
		return nil, err
	}

	dirs, err := f.Readdirnames(-1)

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
func (lpm *LinuxProcessManager) Exists(pid int64) (bool, error) {
	trans := lpm.motor.Transport
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))
	afutil := afero.Afero{Fs: trans.FS()}
	return afutil.Exists(pidPath)
}

func (lpm *LinuxProcessManager) Process(pid int64) (*OSProcess, error) {
	trans := lpm.motor.Transport
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))

	exists, err := lpm.Exists(pid)
	if err != nil {
		return nil, err
	}
	if exists != true {
		return nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist: " + pidPath)
	}

	// parse the cmdline
	cmdlinef, err := trans.File(filepath.Join(pidPath, "cmdline"))
	if err != nil {
		return nil, err
	}
	defer cmdlinef.Close()

	cmdline, err := procfs.ParseProcessCmdline(cmdlinef)
	if err != nil {
		return nil, err
	}

	statusf, err := trans.File(filepath.Join(pidPath, "status"))
	if err != nil {
		return nil, err
	}
	defer statusf.Close()

	status, err := procfs.ParseProcessStatus(statusf)
	if err != nil {
		return nil, err
	}

	process := &OSProcess{
		Pid:        pid,
		Executable: status.Executable,
		State:      status.State,
		Command:    cmdline,
	}

	return process, nil
}

type UnixProcessManager struct {
	motor *motor.Motor
}

func (upm *UnixProcessManager) Name() string {
	return "Unix Process Manager"
}

func (upm *UnixProcessManager) List() ([]*OSProcess, error) {
	c, err := upm.motor.Transport.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
	if err != nil {
		return nil, fmt.Errorf("processes> could not run command")
	}

	entries, err := ParseUnixPsResult(c.Stdout)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("processes", len(entries)).Msg("found processes")

	var ps []*OSProcess
	for i := range entries {
		ps = append(ps, &OSProcess{
			Pid:     entries[i].Pid,
			Command: entries[i].Command,
			State:   "",
		})
	}
	return ps, nil
}

func (upm *UnixProcessManager) Exists(pid int64) (bool, error) {
	return true, nil
	// return false, errors.New("not implemented")
}

func (upm *UnixProcessManager) Process(pid int64) (*OSProcess, error) {
	return nil, errors.New("not implemented")
}
