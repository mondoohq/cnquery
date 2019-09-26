package reboot

import motor "go.mondoo.io/mondoo/motor/motoros"

const LinuxRebootFile = "/var/run/reboot-required"

type LinuxReboot struct {
	Motor *motor.Motor
}

func (s *LinuxReboot) Name() string {
	return "Linux Reboot"
}

func (s *LinuxReboot) RebootRequired() (bool, error) {
	// try to stat the file
	_, err := s.Motor.Transport.FS().Stat(LinuxRebootFile)
	if err != nil {
		return false, nil
	}
	return true, nil
}
