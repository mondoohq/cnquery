package packages

import (
	"fmt"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type SuseUpdateManager struct {
	Motor *motor.Motor
}

func (sum *SuseUpdateManager) Name() string {
	return "Suse Update Manager"
}

func (sum *SuseUpdateManager) Format() string {
	return "suse"
}

func (sum *SuseUpdateManager) List() ([]OperatingSystemUpdate, error) {
	cmd, err := sum.Motor.Transport.RunCommand("zypper --xmlout list-updates -t patch")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return ParseZypperPatches(cmd.Stdout)
}
