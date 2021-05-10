package win

import (
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor/transports"
)

// powershellGetWindowsOSBuild runs a powershell script to retrieve the current version from windows
func powershellGetWindowsOSBuild(t transports.Transport) (*WindowsCurrentVersion, error) {
	pscommand := "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' -Name CurrentBuild, UBR, EditionID | ConvertTo-Json"
	cmd, err := t.RunCommand(powershell.Wrap(pscommand))
	if err != nil {
		return nil, err
	}
	return ParseWinRegistryCurrentVersion(cmd.Stdout)
}
