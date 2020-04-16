package processes

import (
	"errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type OSProcess struct {
	Pid        int64
	Command    string
	Executable string
	State      string
	Uid        int64
}

type OSProcessManager interface {
	Name() string
	Exists(pid int64) (bool, error)
	Process(pid int64) (*OSProcess, error)
	List() ([]*OSProcess, error)
}

func ResolveManager(motor *motor.Motor) (OSProcessManager, error) {
	var pm OSProcessManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: switch to proc based process manager
	// if platform.IsFamily("linux") {
	// case "linux":
	// pm = &LinuxProcManager{motor: motor}
	if platform.IsFamily("unix") {
		pm = &UnixProcessManager{motor: motor, platform: platform}
	} else if platform.IsFamily("windows") {
		pm = &WindowsProcessManager{motor: motor}
	}

	if pm == nil {
		return nil, errors.New("could not detect suitable process manager for platform: " + platform.Name)
	}

	return pm, nil
}
