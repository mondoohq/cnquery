// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
)

type mqlMacosGatekeeperInternal struct {
	lock    sync.Mutex
	fetched bool
	output  string
}

func (m *mqlMacosGatekeeper) fetchStatus() (string, error) {
	if m.fetched {
		return m.output, nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.fetched {
		return m.output, nil
	}

	res, err := NewResource(m.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("spctl --status"),
	})
	if err != nil {
		return "", err
	}
	cmd := res.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", errors.New("spctl --status failed: " + cmd.GetStderr().Data)
	}

	// spctl --status outputs:
	// "assessments enabled" (Gatekeeper on)
	// "assessments disabled" (Gatekeeper off)
	m.output = strings.TrimSpace(cmd.GetStdout().Data)
	m.fetched = true
	return m.output, nil
}

func (m *mqlMacosGatekeeper) status() (string, error) {
	return m.fetchStatus()
}

func (m *mqlMacosGatekeeper) enabled() (bool, error) {
	status, err := m.fetchStatus()
	if err != nil {
		return false, err
	}

	return strings.Contains(status, "assessments enabled"), nil
}
