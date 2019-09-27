package uptime

import (
	"errors"
	"time"

	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/platform"
)

type Uptime interface {
	Name() string
	Duration() (time.Duration, error)
}

func New(motor *motor.Motor) (Uptime, error) {
	var rebootResource Uptime

	pi, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pi.IsFamily(platform.FAMILY_UNIX):
		return &Unix{Motor: motor}, nil
	case pi.IsFamily(platform.FAMILY_WINDOWS):
		return &Windows{Motor: motor}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}

	return rebootResource, nil
}
