package processes

import (
	"errors"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/ssh"
)

type OSProcess struct {
	Pid          int64
	Command      string
	Executable   string
	State        string
	Uid          int64
	SocketInodes []int64
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

	// procfs over ssh is super slow, lets deactivate until we have a faster approach
	disableProcFs := false
	switch motor.Transport.(type) {
	case *ssh.SSHTransport:
		disableProcFs = true
	}

	switch {
	case platform.Runtime == "docker container":
		pm = &DockerTopManager{motor: motor}
	case platform.IsFamily("linux") && !disableProcFs:
		pm = &LinuxProcManager{motor: motor}
	case platform.IsFamily("unix"):
		pm = &UnixProcessManager{motor: motor, platform: platform}
	case platform.IsFamily("windows"):
		pm = &WindowsProcessManager{motor: motor}
	default:
		return nil, errors.New("could not detect suitable process manager for platform: " + platform.Name)
	}

	return pm, nil
}
