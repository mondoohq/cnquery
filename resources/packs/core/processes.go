package core

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/resources/packs/core/processes"
)

func (p *lumiProcess) init(args *lumi.Args) (*lumi.Args, Process, error) {
	pidValue, ok := (*args)["pid"]

	// do not try to resolve the process if we already go all parameters
	// NOTE: this happens for a call like processes.list
	if len(*args) > 2 {
		return args, nil, nil
	}

	// check if additional information is already provided,
	// this let us abort testing if provided by a list
	// _, eok := (*args)["executable"]

	// pid was provided, lets collect the info
	if ok {
		pid, ok := pidValue.(int64)
		if !ok {
			return nil, nil, errors.New("pid has invalid type")
		}

		// lets do minimal IO in initialize
		opm, err := processes.ResolveManager(p.MotorRuntime.Motor)
		if err != nil {
			return nil, nil, errors.New("cannot find process manager")
		}

		// check that the PID exists
		exists, err := opm.Exists(pid)
		if err != nil || !exists {
			return nil, nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist")
		}
	}
	return args, nil, nil
}

func (p *lumiProcess) id() (string, error) {
	pid, err := p.Pid()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(pid, 10), nil
}

func (p *lumiProcess) GetState() (string, error) {
	_, ok := p.Cache.Load("state")
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherProcessInfo(func() {
		err := p.MotorRuntime.Observers.Trigger(p.LumiResource().FieldUID("state"))
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
		err := p.MotorRuntime.Observers.Trigger(p.LumiResource().FieldUID("executable"))
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
		err := p.MotorRuntime.Observers.Trigger(p.LumiResource().FieldUID("command"))
		if err != nil {
			log.Error().Err(err).Msg("[process]> failed to trigger command")
		}
	})

	return "", lumi.NotReadyError{}
}

func (p *lumiProcess) GetFlags() (map[string]interface{}, error) {
	cmd, err := p.Command()
	if err != nil {
		return nil, err
	}

	fs := processes.FlagSet{}
	err = fs.ParseCommand(cmd)
	if err != nil {
		return nil, err
	}
	flags := fs.Map()

	res := map[string]interface{}{}
	for k := range flags {
		res[k] = flags[k]
	}
	return res, nil
}

type ProcessCallbackTrigger func()

func (p *lumiProcess) gatherProcessInfo(fn ProcessCallbackTrigger) error {
	pid, err := p.Pid()
	if err != nil {
		return errors.New("cannot gather pid")
	}

	opm, err := processes.ResolveManager(p.MotorRuntime.Motor)
	if err != nil {
		return errors.New("cannot find process manager")
	}

	process, err := opm.Process(pid)
	if err != nil {
		return errors.New("cannot gather process details")
	}

	sockets := make([]interface{}, len(process.SocketInodes))
	for i := range process.SocketInodes {
		sockets[i] = process.SocketInodes[i]
	}

	now := time.Now().Unix()
	p.Cache.Store("state", &lumi.CacheEntry{Data: process.State, Valid: true, Timestamp: now})
	p.Cache.Store("executable", &lumi.CacheEntry{Data: process.Executable, Valid: true, Timestamp: now})
	p.Cache.Store("command", &lumi.CacheEntry{Data: process.Command, Valid: true, Timestamp: now})
	p.Cache.Store("sockets", &lumi.CacheEntry{Data: sockets, Valid: true, Timestamp: now})

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
	opm, err := processes.ResolveManager(p.MotorRuntime.Motor)
	if opm == nil || err != nil {
		log.Debug().Err(err).Msg("lumi[processes]> could not retrieve process resolver")
		return nil, errors.New("cannot find process manager")
	}

	// retrieve all system processes
	processes, err := opm.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[processes]> could not retrieve process list")
		return nil, fmt.Errorf("could not retrieve process list")
	}
	log.Debug().Int("processes", len(processes)).Msg("lumi[processes]> running processes")

	procs := make([]interface{}, len(processes))
	processesMap := make(map[int64]Process, len(processes))
	socketsMap := map[int64]Process{}

	for i := range processes {
		proc := processes[i]

		lumiProcess, err := p.MotorRuntime.CreateResource("process",
			"pid", proc.Pid,
			"executable", proc.Executable,
			"command", proc.Command,
			"state", proc.State,
		)
		if err != nil {
			return nil, err
		}

		process := lumiProcess.(Process)
		procs[i] = process
		processesMap[proc.Pid] = process

		for i := range proc.SocketInodes {
			socketsMap[proc.SocketInodes[i]] = process
		}
	}

	p.Cache.Store("_map", &lumi.CacheEntry{Data: processesMap})
	p.Cache.Store("_socketsMap", &lumi.CacheEntry{Data: socketsMap})

	// return the processes as new entries
	return procs, nil
}
