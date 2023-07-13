package reboot

import (
	"go.mondoo.com/cnquery/motor/providers/os"
)

const LinuxRebootFile = "/var/run/reboot-required"

// DebianReboot works on Debian and Ubuntu
type DebianReboot struct {
	provider os.OperatingSystemProvider
}

func (s *DebianReboot) Name() string {
	return "Linux Reboot"
}

func (s *DebianReboot) RebootPending() (bool, error) {
	// try to stat the file
	_, err := s.provider.FS().Stat(LinuxRebootFile)
	if err != nil {
		return false, nil
	}
	return true, nil
}
