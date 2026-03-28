// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

func (m *mqlMacosFilevault) status() (string, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("fdesetup status")
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	// fdesetup status outputs lines like:
	// "FileVault is On."
	// "FileVault is Off."
	// "Encryption in progress: Percent completed = 50.0"
	// "Decryption in progress: Percent completed = 50.0"
	output := strings.TrimSpace(string(data))

	// Return the first line which contains the primary status
	lines := strings.SplitN(output, "\n", 2)
	return strings.TrimSpace(lines[0]), nil
}

func (m *mqlMacosFilevault) enabled() (bool, error) {
	status, err := m.status()
	if err != nil {
		return false, err
	}

	// FileVault is considered enabled if it's on or encryption is in progress
	return strings.Contains(status, "FileVault is On") ||
		strings.Contains(status, "Encryption in progress"), nil
}
