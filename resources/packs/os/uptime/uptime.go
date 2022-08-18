package uptime

import (
	"errors"
	"time"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers/os"
)

type Uptime interface {
	Name() string
	Duration() (time.Duration, error)
}

func New(motor *motor.Motor) (Uptime, error) {
	var rebootResource Uptime

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("update manager is not supported for platform: " + pf.Name)
	}

	switch {
	case pf.IsFamily(platform.FAMILY_UNIX):
		return &Unix{provider: osProvider}, nil
	case pf.IsFamily(platform.FAMILY_WINDOWS):
		return &Windows{provider: osProvider}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}

	return rebootResource, nil
}
