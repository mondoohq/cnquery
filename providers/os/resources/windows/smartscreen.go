// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

// Microsoft Defender SmartScreen is configured entirely through policy registry
// values; there is no dedicated cmdlet. We read the well-known policy values for
// the three SmartScreen surfaces (Windows/File Explorer, Microsoft Edge, and
// Microsoft Store apps) in a single PowerShell call and emit a normalized JSON
// object. Missing values serialize as null, which we treat as "not configured".
//
// References:
// https://learn.microsoft.com/en-us/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/

const smartScreenScript = `
function Get-RegVal($path, $name) {
  try { return (Get-ItemProperty -Path $path -Name $name -ErrorAction Stop).$name } catch { return $null }
}
$o = [pscustomobject]@{
  EnableSmartScreen                   = Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' 'EnableSmartScreen'
  ShellSmartScreenLevel               = [string](Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' 'ShellSmartScreenLevel')
  EdgeSmartScreenEnabled              = Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Edge' 'SmartScreenEnabled'
  EdgeSmartScreenPuaEnabled           = Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Edge' 'SmartScreenPuaEnabled'
  EdgePreventOverride                 = Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Edge' 'PreventSmartScreenPromptOverride'
  EdgePreventOverrideForFiles         = Get-RegVal 'HKLM:\SOFTWARE\Policies\Microsoft\Edge' 'PreventSmartScreenPromptOverrideForFiles'
  StoreAppsEnableWebContentEvaluation = Get-RegVal 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppHost' 'EnableWebContentEvaluation'
}
$o | ConvertTo-Json -Compress
`

// SmartScreenSettings holds the raw policy values that drive SmartScreen. The
// numeric toggles use pointers so a missing (null) value is distinguishable
// from an explicit 0.
type SmartScreenSettings struct {
	EnableSmartScreen                   *int64
	ShellSmartScreenLevel               string
	EdgeSmartScreenEnabled              *int64
	EdgeSmartScreenPuaEnabled           *int64
	EdgePreventOverride                 *int64
	EdgePreventOverrideForFiles         *int64
	StoreAppsEnableWebContentEvaluation *int64
}

// enabledFlag reports whether a pointer-backed DWORD toggle is present and set to 1.
func enabledFlag(v *int64) bool { return v != nil && *v == 1 }

// ExplorerEnabled reports whether SmartScreen is enabled for Windows/File Explorer.
func (s *SmartScreenSettings) ExplorerEnabled() bool { return enabledFlag(s.EnableSmartScreen) }

// EdgeEnabled reports whether SmartScreen is enabled for Microsoft Edge.
func (s *SmartScreenSettings) EdgeEnabled() bool { return enabledFlag(s.EdgeSmartScreenEnabled) }

// EdgePuaEnabled reports whether Edge SmartScreen blocks potentially unwanted apps.
func (s *SmartScreenSettings) EdgePuaEnabled() bool { return enabledFlag(s.EdgeSmartScreenPuaEnabled) }

// EdgePreventOverrideEnabled reports whether users are prevented from bypassing
// Edge SmartScreen warnings about sites.
func (s *SmartScreenSettings) EdgePreventOverrideEnabled() bool {
	return enabledFlag(s.EdgePreventOverride)
}

// EdgePreventOverrideForFilesEnabled reports whether users are prevented from
// bypassing Edge SmartScreen warnings about downloads.
func (s *SmartScreenSettings) EdgePreventOverrideForFilesEnabled() bool {
	return enabledFlag(s.EdgePreventOverrideForFiles)
}

// StoreAppsEnabled reports whether SmartScreen web-content evaluation is enabled
// for Microsoft Store apps.
func (s *SmartScreenSettings) StoreAppsEnabled() bool {
	return enabledFlag(s.StoreAppsEnableWebContentEvaluation)
}

// GetSmartScreenSettings reads the SmartScreen policy values from the target.
func GetSmartScreenSettings(p shared.Connection) (*SmartScreenSettings, error) {
	c, err := p.RunCommand(powershell.Encode(smartScreenScript))
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(c.Stdout)
	if err != nil {
		return nil, err
	}
	return ParseSmartScreenSettings(data)
}

func ParseSmartScreenSettings(data []byte) (*SmartScreenSettings, error) {
	var s SmartScreenSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
