// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

func (m *mqlMacosGatekeeper) status() (string, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("spctl --status")
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	// spctl --status outputs:
	// "assessments enabled" (Gatekeeper on)
	// "assessments disabled" (Gatekeeper off)
	return strings.TrimSpace(string(data)), nil
}

func (m *mqlMacosGatekeeper) enabled() (bool, error) {
	status, err := m.status()
	if err != nil {
		return false, err
	}

	return strings.Contains(status, "assessments enabled"), nil
}
