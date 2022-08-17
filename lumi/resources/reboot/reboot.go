package reboot

import (
	"errors"

	"go.mondoo.io/mondoo/motor/providers/os"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
)

type Reboot interface {
	Name() string
	RebootPending() (bool, error)
}

func New(motor *motor.Motor) (Reboot, error) {
	var rebootResource Reboot

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("services manager is not supported for platform: " + pf.Name)
	}

	switch {
	case pf.IsFamily("debian"):
		return &DebianReboot{provider: osProvider}, nil
	case pf.IsFamily("redhat") || pf.Name == "amazonlinux":
		return &RpmNewestKernel{provider: osProvider}, nil
	case pf.IsFamily(platform.FAMILY_WINDOWS):
		return &WinReboot{provider: osProvider}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}

	return rebootResource, nil
}
