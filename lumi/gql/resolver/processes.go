package resolver

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/lumi/resources/processes"
)

func (r *queryResolver) Processes(ctx context.Context) ([]*gql.Process, error) {
	// find suitable package manager
	opm, err := processes.ResolveManager(r.Runtime.Motor)
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

	procs := []*gql.Process{}
	for i := range processes {
		process := processes[i]

		procs = append(procs, &gql.Process{
			Pid:        process.Pid,
			State:      process.State,
			Executable: process.Executable,
			Command:    process.Command,
			Uid:        process.Uid,
		})
	}

	// return the processes as new entries
	return procs, nil
}

func (r *queryResolver) Process(ctx context.Context, pidVal int) (*gql.Process, error) {
	pid := int64(pidVal)
	opm, err := processes.ResolveManager(r.Runtime.Motor)
	if err != nil {
		return nil, errors.New("cannot find process manager")
	}

	// check that the PID exists
	exists, err := opm.Exists(pid)
	if err != nil || exists != true {
		return nil, errors.New("process " + strconv.FormatInt(pid, 10) + " does not exist")
	}

	process, err := opm.Process(pid)
	if err != nil {
		return nil, errors.New("cannot gather process details")
	}

	return &gql.Process{
		Pid:        process.Pid,
		State:      process.State,
		Executable: process.Executable,
		Command:    process.Command,
		Uid:        process.Uid,
	}, nil
}
