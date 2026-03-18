// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/mql/v13/llx"
)

type mqlApparmorInternal struct {
	status  *apparmorStatus
	fetched bool
	lock    sync.Mutex
}

func (a *mqlApparmor) id() (string, error) {
	return "apparmor", nil
}

// apparmorStatus represents the JSON output of apparmor_status --json
type apparmorStatus struct {
	Version   string                        `json:"version"`
	Profiles  map[string]string             `json:"profiles"`
	Processes map[string][]apparmorProcInfo `json:"processes"`
}

type apparmorProcInfo struct {
	Profile string `json:"profile"`
	PID     string `json:"pid"`
	Status  string `json:"status"`
}

func (a *mqlApparmor) fetchStatus() (*apparmorStatus, error) {
	if a.fetched {
		return a.status, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.status, nil
	}

	o, err := CreateResource(a.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("apparmor_status --json"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve apparmor status: " + cmd.Stderr.Data)
	}

	var status apparmorStatus
	if err := json.Unmarshal([]byte(cmd.Stdout.Data), &status); err != nil {
		return nil, err
	}

	a.status = &status
	a.fetched = true
	return a.status, nil
}

func (a *mqlApparmor) version() (string, error) {
	status, err := a.fetchStatus()
	if err != nil {
		return "", err
	}
	return status.Version, nil
}

func (a *mqlApparmor) profiles() ([]any, error) {
	status, err := a.fetchStatus()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(status.Profiles))
	for name, mode := range status.Profiles {
		r, err := CreateResource(a.MqlRuntime, "apparmor.profile", map[string]*llx.RawData{
			"name": llx.StringData(name),
			"mode": llx.StringData(mode),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (a *mqlApparmor) processes() ([]any, error) {
	status, err := a.fetchStatus()
	if err != nil {
		return nil, err
	}

	var res []any
	for executable, procs := range status.Processes {
		for _, p := range procs {
			pid, err := strconv.Atoi(p.PID)
			if err != nil {
				return nil, errors.New("invalid pid " + p.PID)
			}
			r, err := CreateResource(a.MqlRuntime, "apparmor.process", map[string]*llx.RawData{
				"executable": llx.StringData(executable),
				"profile":    llx.StringData(p.Profile),
				"pid":        llx.IntData(int64(pid)),
				"status":     llx.StringData(p.Status),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
	}
	return res, nil
}

func (a *mqlApparmorProfile) id() (string, error) {
	return "apparmor.profile:" + a.Name.Data, nil
}

func (a *mqlApparmorProcess) id() (string, error) {
	return "apparmor.process:" + a.Executable.Data + ":" + strconv.FormatInt(a.Pid.Data, 10), nil
}
