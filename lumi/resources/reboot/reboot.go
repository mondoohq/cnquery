package reboot

import (
	"errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/platform"
)

type Reboot interface {
	Name() string
	RebootRequired() (bool, error)
}

func New(motor *motor.Motor) (Reboot, error) {
	var rebootResource Reboot

	pi, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pi.IsFamily(platform.FAMILY_LINUX):
		return &LinuxReboot{Motor: motor}, nil
	case pi.IsFamily(platform.FAMILY_WINDOWS):
		return &WinReboot{Motor: motor}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}

	return rebootResource, nil
}
