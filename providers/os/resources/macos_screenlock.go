// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"math"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.com/mql/providers/os/connection/shared"
)

type mqlMacosScreenlockInternal struct {
	lock    sync.Mutex
	fetched bool
	ask     bool
	delay   int64
}

// fetch reads the screen-lock settings from the com.apple.screensaver ByHost
// domain for the primary console user. Missing values fail closed: no
// askForPassword means false, and no delay means a large sentinel so that
// "delay <= N" compliance checks do not pass on absent data.
func (m *mqlMacosScreenlock) fetch() error {
	if m.fetched {
		return nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.fetched {
		return nil
	}

	conn, ok := m.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil
	}

	m.ask = strings.TrimSpace(readDefault(conn, "askForPassword")) == "1"

	m.delay = math.MaxInt32
	if v, err := strconv.ParseInt(strings.TrimSpace(readDefault(conn, "askForPasswordDelay")), 10, 64); err == nil {
		m.delay = v
	}

	m.fetched = true
	return nil
}

func readDefault(conn shared.Connection, key string) string {
	cmd, err := conn.RunCommand("defaults -currentHost read com.apple.screensaver " + key)
	if err != nil || cmd.ExitStatus != 0 {
		return ""
	}
	return readAll(cmd.Stdout)
}

func (m *mqlMacosScreenlock) askForPassword() (bool, error) {
	if err := m.fetch(); err != nil {
		return false, err
	}
	return m.ask, nil
}

func (m *mqlMacosScreenlock) askForPasswordDelay() (int64, error) {
	if err := m.fetch(); err != nil {
		return 0, err
	}
	return m.delay, nil
}
