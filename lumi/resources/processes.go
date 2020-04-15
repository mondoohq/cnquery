package resources

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/processes"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

func (p *lumiProcess) init(args *lumi.Args) (*lumi.Args, error) {
	pidValue, ok := (*args)["pid"]

	// check if additional information is already provided,
	// this let us abort testing if provided by a list
	// _, eok := (*args)["executable"]

	// pid was provided, lets collect the info
	if ok {
		pid, ok := pidValue.(int64)
		if !ok {
			return nil, errors.New("pid has invalid type")
		}

		// lets do minimal IO in initialize
		opm, err := resolveOSProcessManager(p.Runtime.Motor)
		if err != nil {
			return nil, errors.New("cannot find process manager")
		}

		// check that the PID exists
		exists, err := opm.Exists(pid)
		if err != nil || exists != true {
			return nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist")
		}
	}
	return args, nil
}

func (p *lumiProcess) id() (string, error) {
	pid, err := p.Pid()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(pid, 10), nil
}

func (p *lumiProcesses) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiProcess) GetState() (string, error) {
	_, ok := p.Cache.Load("state")
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherProcessInfo(func() {
		err := p.Runtime.Observers.Trigger(p.LumiResource().FieldUID("state"))
		if err != nil {
			log.Error().Err(err).Msg("[process]> failed to trigger state")
		}
	})

	return "", lumi.NotReadyError{}
}

func (p *lumiProcess) GetExecutable() (string, error) {
	_, ok := p.Cache.Load("executable")
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherProcessInfo(func() {
		err := p.Runtime.Observers.Trigger(p.LumiResource().FieldUID("executable"))
		if err != nil {
			log.Error().Err(err).Msg("[process]> failed to trigger executable")
		}
	})

	return "", lumi.NotReadyError{}
}

func (p *lumiProcess) GetCommand() (string, error) {
	_, ok := p.Cache.Load("command")
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherProcessInfo(func() {
		err := p.Runtime.Observers.Trigger(p.LumiResource().FieldUID("command"))
		if err != nil {
			log.Error().Err(err).Msg("[process]> failed to trigger command")
		}
	})

	return "", lumi.NotReadyError{}
}

type ProcessCallbackTrigger func()

func (p *lumiProcess) gatherProcessInfo(fn ProcessCallbackTrigger) error {
	pid, err := p.Pid()
	if err != nil {
		return errors.New("cannot gather pid")
	}

	opm, err := resolveOSProcessManager(p.Runtime.Motor)
	if err != nil {
		return errors.New("cannot find process manager")
	}

	process, err := opm.Process(pid)
	if err != nil {
		return errors.New("cannot gather process details")
	}

	p.Cache.Store("state", &lumi.CacheEntry{Data: process.State, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("executable", &lumi.CacheEntry{Data: process.Executable, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("command", &lumi.CacheEntry{Data: process.Command, Valid: true, Timestamp: time.Now().Unix()})

	// call callback trigger
	if fn != nil {
		fn()
	}

	return nil
}

func (p *lumiProcesses) id() (string, error) {
	return "processes", nil
}

func (p *lumiProcesses) GetList() ([]interface{}, error) {

	// find suitable package manager
	opm, err := resolveOSProcessManager(p.Runtime.Motor)
	if opm == nil || err != nil {
		log.Warn().Err(err).Msg("lumi[processes]> could not retrieve process list")
		return nil, errors.New("cannot find process manager")
	}

	// retrieve all system processes
	processes, err := opm.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[processes]> could not retrieve process list")
		return nil, fmt.Errorf("could not retrieve process list")
	}
	log.Debug().Int("processes", len(processes)).Msg("lumi[processes]> running processes")

	procs := []interface{}{}
	for i := range processes {
		proc := processes[i]

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["pid"] = proc.Pid

		// TODO: harmonize with the mapping for individual packages
		args["executable"] = proc.Executable
		args["command"] = proc.Command
		args["state"] = proc.State

		e, err := newProcess(p.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Int64("process", proc.Pid).Msg("lumi[processes]> could not create process resource")
			continue
		}

		procs = append(procs, e.(Process))
	}

	// return the processes as new entries
	return procs, nil
}

func resolveOSProcessManager(motor *motor.Motor) (OSProcessManager, error) {
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
	pidPath := filepath.Join("/proc", strconv.FormatInt(pid, 10))

	fs := lpm.motor.Transport.FS()
	afs := &afero.Afero{Fs: fs}
	return afs.Exists(pidPath)
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

	entries, err := processes.ParseUnixPsResult(c.Stdout)
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
