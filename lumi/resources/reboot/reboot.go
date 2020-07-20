package reboot

import (
	"errors"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
)

type Reboot interface {
	Name() string
	RebootPending() (bool, error)
}

func New(motor *motor.Motor) (Reboot, error) {
	var rebootResource Reboot

	pi, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pi.IsFamily("debian"):
		return &DebianReboot{Motor: motor}, nil
	case pi.Name == "amazonlinux":
		fallthrough
	case pi.IsFamily("redhat"):
		return &RpmNewestKernel{Motor: motor}, nil
	case pi.IsFamily(platform.FAMILY_WINDOWS):
		return &WinReboot{Motor: motor}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}

	return rebootResource, nil
}
