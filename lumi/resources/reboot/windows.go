package reboot

import (
	"fmt"
	"io/ioutil"
	"strings"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

const (
	WindowsTestComponentServicesReboot = "Test-Path -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Component Based Servicing\\RebootPending'"
	WindowsTestWsusReboot              = "Test-Path -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\WindowsUpdate\\Auto Update\\RebootRequired'"
)

// WinReboot checks if the windows instance requires a reboot
// Excellent resources:
// https://blogs.technet.microsoft.com/heyscriptingguy/2013/06/10/determine-pending-reboot-statuspowershell-style-part-1/
// https://blogs.technet.microsoft.com/heyscriptingguy/2013/06/11/determine-pending-reboot-statuspowershell-style-part-2/
// Brian Wilhite powershell implementation:
// https://github.com/bcwilhite/PendingReboot
type WinReboot struct {
	Motor *motor.Motor
}

func (s *WinReboot) Name() string {
	return "Windows Reboot"
}

func (s *WinReboot) RebootRequired() (bool, error) {
	isRebootrequired := false

	// Query the Component Based Servicing Reg Key
	cmd, err := s.Motor.Transport.RunCommand(fmt.Sprintf("powershell -c \"%s\"", WindowsTestComponentServicesReboot))
	if err != nil {
		return false, err
	}

	content, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(strings.ToLower(string(content))) == "true" {
		isRebootrequired = true
	}

	// Query WUAU from the registry
	cmd, err = s.Motor.Transport.RunCommand(fmt.Sprintf("powershell -c \"%s\"", WindowsTestWsusReboot))
	if err != nil {
		return false, err
	}

	content, err = ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(strings.ToLower(string(content))) == "true" {
		isRebootrequired = true
	}

	// Query PendingFileRenameOperations from the registry
	// Note: we are not using it since its also used by non-OS specific apps

	return isRebootrequired, nil
}
