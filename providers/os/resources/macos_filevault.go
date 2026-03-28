// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
)

type mqlMacosFilevaultInternal struct {
	lock    sync.Mutex
	fetched bool
	output  string
}

func (m *mqlMacosFilevault) fetchStatus() (string, error) {
	if m.fetched {
		return m.output, nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.fetched {
		return m.output, nil
	}

	res, err := NewResource(m.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("fdesetup status"),
	})
	if err != nil {
		return "", err
	}
	cmd := res.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", errors.New("fdesetup status failed: " + cmd.GetStderr().Data)
	}

	// fdesetup status outputs lines like:
	// "FileVault is On."
	// "FileVault is Off."
	// "Encryption in progress: Percent completed = 50.0"
	// "Decryption in progress: Percent completed = 50.0"
	output := strings.TrimSpace(cmd.GetStdout().Data)
	lines := strings.SplitN(output, "\n", 2)

	m.output = strings.TrimSpace(lines[0])
	m.fetched = true
	return m.output, nil
}

func (m *mqlMacosFilevault) status() (string, error) {
	return m.fetchStatus()
}

func (m *mqlMacosFilevault) enabled() (bool, error) {
	status, err := m.fetchStatus()
	if err != nil {
		return false, err
	}

	return strings.Contains(status, "FileVault is On") ||
		strings.Contains(status, "Encryption in progress"), nil
}

func (m *mqlMacosFilevault) runFdesetup(subcmd string) (string, error) {
	res, err := NewResource(m.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("fdesetup " + subcmd),
	})
	if err != nil {
		return "", err
	}
	cmd := res.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", errors.New("fdesetup " + subcmd + " failed: " + cmd.GetStderr().Data)
	}
	return strings.TrimSpace(cmd.GetStdout().Data), nil
}

func (m *mqlMacosFilevault) hasPersonalRecoveryKey() (bool, error) {
	enabled, err := m.enabled()
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}
	// fdesetup haspersonalrecoverykey outputs "true" or "false"
	output, err := m.runFdesetup("haspersonalrecoverykey")
	if err != nil {
		return false, err
	}
	return output == "true", nil
}

func (m *mqlMacosFilevault) hasInstitutionalRecoveryKey() (bool, error) {
	enabled, err := m.enabled()
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}
	// fdesetup hasinstitutionalrecoverykey outputs "true" or "false"
	output, err := m.runFdesetup("hasinstitutionalrecoverykey")
	if err != nil {
		return false, err
	}
	return output == "true", nil
}

func (m *mqlMacosFilevault) users() ([]any, error) {
	enabled, err := m.enabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return []any{}, nil
	}
	// fdesetup list outputs lines like:
	// "user1,85632A00-1234-5678-ABCD-123456789ABC"
	// "user2,95632A00-1234-5678-ABCD-123456789ABC"
	output, err := m.runFdesetup("list")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []any{}, nil
	}

	var users []any
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line is "username,UUID" — extract the username
		parts := strings.SplitN(line, ",", 2)
		users = append(users, parts[0])
	}

	return users, nil
}
