package reboot

import "go.mondoo.io/mondoo/motor"

const LinuxRebootFile = "/var/run/reboot-required"

// DebianReboot works on Debian and Ubuntu
type DebianReboot struct {
	Motor *motor.Motor
}

func (s *DebianReboot) Name() string {
	return "Linux Reboot"
}

func (s *DebianReboot) RebootPending() (bool, error) {
	// try to stat the file
	_, err := s.Motor.Transport.FS().Stat(LinuxRebootFile)
	if err != nil {
		return false, nil
	}
	return true, nil
}
