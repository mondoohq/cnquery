// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
)

func (t *mqlSystemdTimesyncd) id() (string, error) {
	return "systemd.timesyncd", nil
}

func (t *mqlSystemdTimesyncd) active() (bool, error) {
	return isSystemdUnitActive(t.MqlRuntime, "systemd-timesyncd"), nil
}

type timesyncdState struct {
	synchronized     bool
	servers          []string
	fallbackServers  []string
	serverName       string
	serverAddress    string
	pollIntervalUSec int64
	leapStatus       string
}

func (t *mqlSystemdTimesyncd) resolveState() (*timesyncdState, error) {
	if t.cachedState != nil {
		return t.cachedState, nil
	}
	state := &timesyncdState{}

	// Synchronized state lives in `timedatectl show` (the dbus-exposed
	// org.freedesktop.timedate1 properties), not in show-timesync.
	if stdout, ok, err := runSystemctl(t.MqlRuntime, "timedatectl show --no-pager"); err != nil {
		return nil, err
	} else if ok {
		props := parseSystemdShowOutput(stdout)
		state.synchronized = props["NTPSynchronized"] == "yes"
	}

	// Per-server state comes from show-timesync (org.freedesktop.timesync1).
	if stdout, ok, err := runSystemctl(t.MqlRuntime, "timedatectl show-timesync --no-pager"); err != nil {
		return nil, err
	} else if ok {
		props := parseSystemdShowOutput(stdout)
		state.servers = strings.Fields(props["SystemNTPServers"])
		// Older systemd versions emit `LinkNTPServers` instead of including
		// DHCP-provided servers in SystemNTPServers; fold them in so the
		// effective list is what users get.
		state.servers = append(state.servers, strings.Fields(props["LinkNTPServers"])...)
		state.fallbackServers = strings.Fields(props["FallbackNTPServers"])
		state.serverName = props["ServerName"]
		state.serverAddress = props["ServerAddress"]
		state.leapStatus = props["LeapStatus"]
		if v, err := strconv.ParseInt(strings.TrimSpace(props["PollIntervalUSec"]), 10, 64); err == nil {
			state.pollIntervalUSec = v
		}
	}

	t.cachedState = state
	return state, nil
}

func (t *mqlSystemdTimesyncd) synchronized() (bool, error) {
	s, err := t.resolveState()
	if err != nil {
		return false, err
	}
	return s.synchronized, nil
}

func (t *mqlSystemdTimesyncd) servers() ([]any, error) {
	s, err := t.resolveState()
	if err != nil {
		return nil, err
	}
	return stringsToAny(s.servers), nil
}

func (t *mqlSystemdTimesyncd) fallbackServers() ([]any, error) {
	s, err := t.resolveState()
	if err != nil {
		return nil, err
	}
	return stringsToAny(s.fallbackServers), nil
}

func (t *mqlSystemdTimesyncd) serverName() (string, error) {
	s, err := t.resolveState()
	if err != nil {
		return "", err
	}
	return s.serverName, nil
}

func (t *mqlSystemdTimesyncd) serverAddress() (string, error) {
	s, err := t.resolveState()
	if err != nil {
		return "", err
	}
	return s.serverAddress, nil
}

func (t *mqlSystemdTimesyncd) pollIntervalUSec() (int64, error) {
	s, err := t.resolveState()
	if err != nil {
		return 0, err
	}
	return s.pollIntervalUSec, nil
}

func (t *mqlSystemdTimesyncd) leapStatus() (string, error) {
	s, err := t.resolveState()
	if err != nil {
		return "", err
	}
	return s.leapStatus, nil
}

type mqlSystemdTimesyncdInternal struct {
	cachedState *timesyncdState
}
