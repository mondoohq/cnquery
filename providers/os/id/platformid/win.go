// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package platformid

import (
	"io"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

const wmiMachineIDQuery = "SELECT UUID FROM Win32_ComputerSystemProduct"

func PowershellWindowsMachineId(conn shared.Connection) (string, error) {
	cmd, err := conn.RunCommand("powershell -c \"Get-WmiObject -Query '" + wmiMachineIDQuery + "' | Select-Object -ExpandProperty UUID\"")
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}
	guid := string(data)
	return guid, nil
}

type WinIdProvider struct {
	connection shared.Connection
}

func (p *WinIdProvider) Name() string {
	return "Windows Machine ID"
}

func (p *WinIdProvider) ID() (string, error) {
	return windowsMachineId(p.connection)
}
