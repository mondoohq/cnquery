// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

const (
	HotpatchPackage = "Hotpatch Enrollment Package"
)

type WindowsHotpatch struct {
	Name                              string `json:"Name"`
	HotPatchTableSize                 string `json:"HotPatchTableSize"`
	EnableVirtualizationBasedSecurity string `json:"EnableVirtualizationBasedSecurity"`
}

func ParseWinRegistryHotpatch(r io.Reader) (bool, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return false, err
	}

	var hotpatch WindowsHotpatch
	err = json.Unmarshal(data, &hotpatch)
	if err != nil {
		return false, err
	}
	log.Debug().Interface("Hotpatch", hotpatch).Msg("Parsed hotpatch information")

	return hotpatch.Name == HotpatchPackage && hotpatch.EnableVirtualizationBasedSecurity == "1" && hotpatch.HotPatchTableSize != "0", nil
}

// https://learn.microsoft.com/en-us/windows-server/get-started/hotpatch
// https://learn.microsoft.com/en-us/windows-server/get-started/enable-hotpatch-azure-edition

// powershellGetWindowsHotpatch runs a powershell script to determine whether hotpatching is enabled on the system.
func powershellGetWindowsHotpatch(conn shared.Connection, arch string) (bool, error) {
	// FIXME: for windows 2025 this might be arm64
	pscommand := `
$info = Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Update\TargetingInfo\DynamicInstalled\Hotpatch.` + strings.ToLower(arch) + `' -Name Name
$sysInfo = Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\DeviceGuard' -Name EnableVirtualizationBasedSecurity
$hotpatch = Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management' -Name HotPatchTableSize
$sysInfo | Add-Member -MemberType NoteProperty -Name Name -Value $info.Name
$hotpatch | Add-Member -MemberType NoteProperty -Name HotPatchTableSize -Value $hotpatch.HotPatchTableSize
$sysInfo | Select-Object Name, EnableVirtualizationBasedSecurity, HotPatchTableSize | ConvertTo-Json
`

	log.Debug().Msg("checking Windows hotpatch runtime")
	cmd, err := conn.RunCommand(powershell.Encode(pscommand))
	if err != nil {
		log.Debug().Err(err).Msg("could not run powershell command to get hotpatch information")
		// Don't return an error here, as it is expected that this key may not exist
		return false, nil
	}
	return ParseWinRegistryHotpatch(cmd.Stdout)
}
