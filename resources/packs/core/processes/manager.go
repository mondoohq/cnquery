package processes

import (
	"errors"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/ssh"
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

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("process manager is not supported for platform: " + pf.Name)
	}

	// procfs over ssh is super slow, lets deactivate until we have a faster approach
	disableProcFs := false
	switch motor.Provider.(type) {
	case *ssh.Provider:
		disableProcFs = true
	}

	switch {
	case pf.Runtime == "docker container":
		pm = &DockerTopManager{provider: osProvider}
	case pf.IsFamily("linux") && !disableProcFs:
		pm = &LinuxProcManager{provider: osProvider}
	case pf.IsFamily("unix"):
		pm = &UnixProcessManager{provider: osProvider, platform: pf}
	case pf.IsFamily("windows"):
		pm = &WindowsProcessManager{provider: osProvider}
	default:
		return nil, errors.New("could not detect suitable process manager for platform: " + pf.Name)
	}

	return pm, nil
}
