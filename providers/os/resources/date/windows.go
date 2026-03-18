// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

type windowsDateResult struct {
	DateTime string `json:"DateTime"`
	Timezone string `json:"Timezone"`
}

// PowerShell command that returns both the current UTC time and the system timezone ID.
const windowsDateCmd = `@{DateTime=(Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ');Timezone=(Get-TimeZone).Id} | ConvertTo-Json`

type Windows struct {
	conn shared.Connection
}

func (w *Windows) Name() string {
	return "Windows Date"
}

func (w *Windows) Get() (*Result, error) {
	if !w.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return &Result{Timezone: "UTC"}, nil
	}

	cmd, err := w.conn.RunCommand(powershell.Wrap(windowsDateCmd))
	if err != nil {
		return nil, fmt.Errorf("failed to get system date: %w", err)
	}

	return w.parse(cmd.Stdout)
}

func (w *Windows) parse(r io.Reader) (*Result, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read date output: %w", err)
	}

	var res windowsDateResult
	if err := json.Unmarshal(content, &res); err != nil {
		return nil, fmt.Errorf("failed to parse date output: %w", err)
	}

	t, err := time.Parse(time.RFC3339, res.DateTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse datetime %q: %w", res.DateTime, err)
	}

	// Windows returns IANA-compatible timezone IDs on modern systems
	loc, err := time.LoadLocation(res.Timezone)
	if err != nil {
		// Fall back to returning UTC time with the Windows timezone ID
		return &Result{
			Time:     &t,
			Timezone: res.Timezone,
		}, nil
	}

	locT := t.In(loc)
	return &Result{
		Time:     &locT,
		Timezone: res.Timezone,
	}, nil
}
