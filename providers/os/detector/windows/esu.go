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

type WindowsESUStatus struct {
	SubscriptionEligible bool `json:"SubscriptionEligible"`
	LicenseActivated     bool `json:"LicenseActivated"`
}

func (e WindowsESUStatus) ESUEnabled() bool {
	return e.SubscriptionEligible || e.LicenseActivated
}

func ParseWindowsESUStatus(r io.Reader) (*WindowsESUStatus, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &WindowsESUStatus{}, nil
	}

	var status WindowsESUStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("ESUStatus", status).Msg("parsed Windows ESU status")

	return &status, nil
}

// powershellGetWindowsESUStatus checks for Windows 10 ESU enrollment via PowerShell.
// It checks both subscription-based ESU (registry key) and MAK-activated ESU (WMI license).
func powershellGetWindowsESUStatus(conn shared.Connection) (*WindowsESUStatus, error) {
	pscommand := `
$result = @{ SubscriptionEligible = $false; LicenseActivated = $false }
$esuPath = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\SoftwareProtectionPlatform\ESU'
if (Test-Path $esuPath) {
    $props = Get-ItemProperty -Path $esuPath -ErrorAction SilentlyContinue
    if ($props.Win10CommercialW365ESUEligible -eq 1) {
        $result.SubscriptionEligible = $true
    }
}
$esuProducts = Get-CimInstance -ClassName SoftwareLicensingProduct -Filter "Name LIKE '%ESU%' AND LicenseStatus = 1" -ErrorAction SilentlyContinue
if ($esuProducts) { $result.LicenseActivated = $true }
$result | ConvertTo-Json -Compress
`

	log.Debug().Msg("checking Windows 10 ESU status")
	cmd, err := conn.RunCommand(powershell.Encode(pscommand))
	if err != nil {
		log.Debug().Err(err).Msg("could not run powershell command to get ESU status")
		return &WindowsESUStatus{}, nil
	}
	return ParseWindowsESUStatus(cmd.Stdout)
}
