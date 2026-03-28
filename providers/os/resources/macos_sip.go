// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

func (m *mqlMacosSip) status() (string, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("csrutil status")
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	// csrutil status outputs:
	// "System Integrity Protection status: enabled."
	// "System Integrity Protection status: disabled."
	// May also include individual configuration flags on separate lines
	output := strings.TrimSpace(string(data))

	// Return the first line which contains the primary status
	lines := strings.SplitN(output, "\n", 2)
	return strings.TrimSpace(lines[0]), nil
}

func (m *mqlMacosSip) enabled() (bool, error) {
	status, err := m.status()
	if err != nil {
		return false, err
	}

	return strings.Contains(status, "enabled"), nil
}
