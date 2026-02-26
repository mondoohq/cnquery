// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

type IntuneDeviceInfo struct {
	EnrollmentGUID string `json:"EnrollmentGUID"`
	EntDMID        string `json:"EntDMID"`
}

func ParseIntuneDeviceID(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	// If the output is empty, the device is not enrolled in Intune
	if len(data) == 0 {
		return "", nil
	}

	var info IntuneDeviceInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		return "", err
	}
	log.Debug().Str("EntDMID", info.EntDMID).Msg("parsed Intune device information")

	return info.EntDMID, nil
}

// powershellGetIntuneDeviceID runs a powershell script to retrieve the Intune device ID from enrolled Windows clients.
func powershellGetIntuneDeviceID(conn shared.Connection) (string, error) {
	pscommand := `Get-ChildItem -Path 'HKLM:\SOFTWARE\Microsoft\Enrollments\' | ForEach-Object { $dmClient = Join-Path $_.PSPath 'DMClient\MS DM Server'; if (Test-Path $dmClient) { $props = Get-ItemProperty $dmClient -ErrorAction SilentlyContinue; if ($props.EntDMID) { [PSCustomObject]@{ EnrollmentGUID = $_.PSChildName; EntDMID = $props.EntDMID } } } } | Select-Object -First 1 | ConvertTo-Json -Compress`

	log.Debug().Msg("checking Intune device ID")
	cmd, err := conn.RunCommand(powershell.Encode(pscommand))
	if err != nil {
		log.Debug().Err(err).Msg("could not run powershell command to get Intune device ID")
		return "", nil
	}
	return ParseIntuneDeviceID(cmd.Stdout)
}
