// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
	"sync"
)

func (t *mqlSystemdTimesyncd) id() (string, error) {
	return "systemd.timesyncd", nil
}

func (t *mqlSystemdTimesyncd) active() (bool, error) {
	return isSystemdUnitActive(t.MqlRuntime, "systemd-timesyncd")
}

// parseTimedatectlStatusSynchronized extracts the synchronization state from
// the human-readable `timedatectl status` output, used as a fallback on
// systemd versions that lack the `timedatectl show` verb (< 239). The
// relevant line looks like:
//
//	System clock synchronized: yes
//
// and is backed by the same org.freedesktop.timedate1 NTPSynchronized
// property that `timedatectl show` exposes on newer systemd.
func parseTimedatectlStatusSynchronized(stdout string) bool {
	for _, line := range strings.Split(stdout, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if strings.TrimSpace(key) != "System clock synchronized" {
			continue
		}
		return strings.EqualFold(strings.TrimSpace(value), "yes")
	}
	return false
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
	if t.fetched {
		return t.cachedState, nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.fetched {
		return t.cachedState, nil
	}
	state := &timesyncdState{
		// Default to an empty slice so callers that pass these through
		// stringsToAny() get []any{} rather than nil when the binary is
		// missing or both NTP server properties are absent.
		servers:         []string{},
		fallbackServers: []string{},
	}

	// Synchronized state lives in the `NTPSynchronized` property of the
	// org.freedesktop.timedate1 dbus interface. On systemd >= 239 this is
	// exposed via `timedatectl show`; that verb does not exist on older
	// systemd (e.g. v237 on Ubuntu 18.04), where `timedatectl show` exits
	// non-zero and returns nothing. In that case fall back to parsing the
	// human-readable `timedatectl status` output, whose "System clock
	// synchronized: yes/no" line is backed by the same property.
	if stdout, ok, err := runSystemctl(t.MqlRuntime, "timedatectl show --no-pager"); err != nil {
		return nil, err
	} else if ok {
		props := parseSystemdShowOutput(stdout)
		state.synchronized = props["NTPSynchronized"] == "yes"
	} else if stdout, ok, err := runSystemctl(t.MqlRuntime, "timedatectl status --no-pager"); err != nil {
		return nil, err
	} else if ok {
		state.synchronized = parseTimedatectlStatusSynchronized(stdout)
	}

	// Per-server state comes from show-timesync (org.freedesktop.timesync1).
	if stdout, ok, err := runSystemctl(t.MqlRuntime, "timedatectl show-timesync --no-pager"); err != nil {
		return nil, err
	} else if ok {
		props := parseSystemdShowOutput(stdout)
		servers := strings.Fields(props["SystemNTPServers"])
		// Older systemd versions emit `LinkNTPServers` instead of including
		// DHCP-provided servers in SystemNTPServers; fold them in so the
		// effective list is what users get.
		servers = append(servers, strings.Fields(props["LinkNTPServers"])...)
		if servers != nil {
			state.servers = servers
		}
		if fb := strings.Fields(props["FallbackNTPServers"]); fb != nil {
			state.fallbackServers = fb
		}
		state.serverName = props["ServerName"]
		state.serverAddress = props["ServerAddress"]
		state.leapStatus = props["LeapStatus"]
		if v, err := strconv.ParseInt(strings.TrimSpace(props["PollIntervalUSec"]), 10, 64); err == nil {
			state.pollIntervalUSec = v
		}
	}

	t.fetched = true
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
	fetched     bool
	lock        sync.Mutex
}
