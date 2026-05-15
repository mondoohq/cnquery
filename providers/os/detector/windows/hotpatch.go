// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

const (
	HotpatchPackage = "Hotpatch Enrollment Package"
)

// isClientOS returns true if the platform's product-type indicates a workstation (Windows client).
func isClientOS(pf *inventory.Platform) bool {
	return pf.Labels["windows.mondoo.com/product-type"] == "1"
}

// WindowsClientHotpatch holds the registry values relevant for client (Win11) hotpatch detection.
type WindowsClientHotpatch struct {
	AllowRebootlessUpdates            string `json:"AllowRebootlessUpdates"`
	EnableVirtualizationBasedSecurity string `json:"EnableVirtualizationBasedSecurity"`
}

// ParseWinRegistryClientHotpatch checks whether AllowRebootlessUpdates and VBS are both enabled.
func ParseWinRegistryClientHotpatch(r io.Reader) (bool, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return false, err
	}

	var hotpatch WindowsClientHotpatch
	err = json.Unmarshal(data, &hotpatch)
	if err != nil {
		return false, err
	}
	log.Debug().Interface("ClientHotpatch", hotpatch).Msg("Parsed client hotpatch information")

	return hotpatch.AllowRebootlessUpdates == "1" && hotpatch.EnableVirtualizationBasedSecurity == "1", nil
}

// hotpatchSupported checks whether the given platform meets the prerequisites
// for hotpatching:
//   - Windows Server 2022+ (build 20348+, product-type "2" or "3")
//   - Windows 11 Enterprise / Education 24H2+ (product-type "1"):
//   - x64 (AMD64/Intel): build 26100.2033+
//   - arm64: build 26100.4929+
//
// Client hotpatch is license-gated, not just build-gated: the eligible SKUs
// (Windows 11 Enterprise E3/E5, Education A3/A5, M365 F3, M365 Business Premium,
// Windows 365 Enterprise) all activate Windows 11 Enterprise or Education on
// the device. A Pro / Home / Pro Education box can never receive hotpatches
// even when AllowRebootlessUpdates + VBS are set in the registry — a config
// that can leak in via a misapplied GPO/Intune policy. Without the edition
// guard we falsely tag those assets `hotpatch=true` and hotpatch-only KBs
// surface as findings (real example: KB5089466 on a Win11 Pro AMD64 box).
//
// References:
//   - https://learn.microsoft.com/en-us/intune/device-updates/windows/hotpatch (eligible license list, 2026-04-29)
//   - https://learn.microsoft.com/en-us/windows/deployment/windows-autopatch/manage/windows-autopatch-hotpatch-updates (Autopatch eligibility, 2026-03-06)
//   - https://learn.microsoft.com/en-us/windows/client-management/hotpatch (technical preconditions: build, UBR, VBS, ARM64 CHPE)
func hotpatchSupported(pf *inventory.Platform) bool {
	buildNumber, err := strconv.Atoi(pf.Version)
	if err != nil {
		log.Error().Err(err).Msg("could not parse windows build number")
		return false
	}
	log.Debug().Int("buildNumber", buildNumber).Msg("parsed windows build number")

	productType := pf.Labels["windows.mondoo.com/product-type"]
	switch productType {
	case "1": // Workstation (Windows client)
		if !isHotpatchEligibleClientEdition(pf.Title) {
			log.Debug().Str("title", pf.Title).Msg("windows client edition is not hotpatch-eligible")
			return false
		}
		if buildNumber > 26100 {
			return true
		}
		if buildNumber < 26100 {
			return false
		}
		ubr, err := strconv.Atoi(pf.Build)
		if err != nil {
			log.Error().Err(err).Msg("could not parse windows UBR")
			return false
		}
		log.Debug().Int("ubr", ubr).Str("arch", pf.Arch).Msg("parsed windows UBR for client hotpatch check")
		minUBR := 2033
		if strings.EqualFold(pf.Arch, "arm64") {
			minUBR = 4929
		}
		return ubr >= minUBR
	case "2", "3": // Domain Controller or Server
		return buildNumber >= 20348
	default:
		return false
	}
}

// isHotpatchEligibleClientEdition reports whether `title` (the human-readable
// Windows edition string from platform detection, e.g. "Windows 11 Enterprise"
// or "Windows 11 Pro") corresponds to a SKU that can receive client hotpatches.
//
// Eligibility is license-gated per Microsoft Intune docs (2026-04-29) but every
// eligible license activates Enterprise or Education on the device. The edition
// string is the closest runtime signal we can read without inspecting tenant
// licensing. Empty Title → refuse, since we can't tell what's running and
// hotpatch is the failure-mode-with-false-positives direction.
//
// "Pro Education" is a distinct SKU in the Pro family (not Education A3/A5) so
// it's rejected explicitly before the broader "education" substring match.
func isHotpatchEligibleClientEdition(title string) bool {
	t := strings.ToLower(title)
	if t == "" {
		return false
	}
	if strings.Contains(t, "pro education") {
		return false
	}
	return strings.Contains(t, "enterprise") || strings.Contains(t, "education")
}

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
// https://learn.microsoft.com/en-us/windows/client-management/hotpatch

// powershellGetWindowsClientHotpatch queries the client-specific AllowRebootlessUpdates policy and VBS.
func powershellGetWindowsClientHotpatch(conn shared.Connection) (bool, error) {
	pscommand := `
$rebootless = Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\PolicyManager\current\device\Update' -Name AllowRebootlessUpdates -ErrorAction SilentlyContinue
$sysInfo = Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\DeviceGuard' -Name EnableVirtualizationBasedSecurity -ErrorAction SilentlyContinue
$result = @{}
if ($rebootless) { $result.AllowRebootlessUpdates = [string]$rebootless.AllowRebootlessUpdates }
if ($sysInfo) { $result.EnableVirtualizationBasedSecurity = [string]$sysInfo.EnableVirtualizationBasedSecurity }
$result | ConvertTo-Json
`

	log.Debug().Msg("checking Windows client hotpatch runtime")
	cmd, err := conn.RunCommand(powershell.Encode(pscommand))
	if err != nil {
		log.Debug().Err(err).Msg("could not run powershell command to get client hotpatch information")
		return false, nil
	}
	return ParseWinRegistryClientHotpatch(cmd.Stdout)
}

// powershellGetWindowsServerHotpatch queries the server-specific hotpatch enrollment, VBS and HotPatchTableSize.
func powershellGetWindowsServerHotpatch(conn shared.Connection, arch string) (bool, error) {
	// FIXME: for windows 2025 this might be arm64
	pscommand := `
$info = Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Update\TargetingInfo\DynamicInstalled\Hotpatch.` + strings.ToLower(arch) + `' -Name Name
$sysInfo = Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\DeviceGuard' -Name EnableVirtualizationBasedSecurity
$hotpatch = Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management' -Name HotPatchTableSize
$sysInfo | Add-Member -MemberType NoteProperty -Name Name -Value $info.Name
$hotpatch | Add-Member -MemberType NoteProperty -Name HotPatchTableSize -Value $hotpatch.HotPatchTableSize
$sysInfo | Select-Object Name, EnableVirtualizationBasedSecurity, HotPatchTableSize | ConvertTo-Json
`

	log.Debug().Msg("checking Windows server hotpatch runtime")
	cmd, err := conn.RunCommand(powershell.Encode(pscommand))
	if err != nil {
		log.Debug().Err(err).Msg("could not run powershell command to get hotpatch information")
		return false, nil
	}
	return ParseWinRegistryHotpatch(cmd.Stdout)
}

// powershellGetWindowsHotpatch runs a powershell script to determine whether hotpatching is enabled on the system.
// Hotpatching is supported on Windows Server 2022+ and Windows 11 Enterprise 24H2+.
func powershellGetWindowsHotpatch(conn shared.Connection, pf *inventory.Platform) (bool, error) {
	if isClientOS(pf) {
		return powershellGetWindowsClientHotpatch(conn)
	}
	return powershellGetWindowsServerHotpatch(conn, pf.Arch)
}
