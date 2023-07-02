package mount

import (
	"errors"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/os"
)

type MountPoint struct {
	Device     string
	MountPoint string
	FSType     string
	Options    map[string]string
}

type OperatingSystemMountManager interface {
	Name() string
	List() ([]MountPoint, error)
}

func ResolveManager(motor *motor.Motor) (OperatingSystemMountManager, error) {
	var mm OperatingSystemMountManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("mount manager is not supported for platform: " + pf.Name)
	}

	for i := range pf.Family {
		if pf.Family[i] == "linux" {
			mm = &LinuxMountManager{provider: osProvider}
			break
		} else if pf.Family[i] == "unix" {
			mm = &UnixMountManager{provider: osProvider}
			break
		}
	}

	if mm == nil {
		return nil, errors.New("could not detect suitable mount manager for platform: " + pf.Name)
	}

	return mm, nil
}

type LinuxMountManager struct {
	provider os.OperatingSystemProvider
}

func (s *LinuxMountManager) Name() string {
	return "Linux Mount Manager"
}

func (s *LinuxMountManager) List() ([]MountPoint, error) {
	// TODO: not working via docker yet
	// // try /proc
	// f, err := s.motor.Provider.File("/proc/mount")
	// if err == nil {
	// 	defer f.Close()
	// 	return ParseLinuxProcMount(f), nil
	// }

	if s.provider.Capabilities().HasCapability(providers.Capability_RunCommand) {
		cmd, err := s.provider.RunCommand("mount")
		if err != nil {
			return nil, errors.Join(err, errors.New("could not read mounts"))
		}
		return ParseLinuxMountCmd(cmd.Stdout), nil
	} else if s.provider.Capabilities().HasCapability(providers.Capability_File) {
		return mountsFromFSLinux(s.provider.FS())
	}

	return nil, errors.New("mount not supported for provided transport")
}

type UnixMountManager struct {
	provider os.OperatingSystemProvider
}

func (s *UnixMountManager) Name() string {
	return "Unix Mount Manager"
}

func (s *UnixMountManager) List() ([]MountPoint, error) {
	cmd, err := s.provider.RunCommand("mount")
	if err != nil {
		return nil, errors.Join(err, errors.New("could not read package list"))
	}

	return ParseUnixMountCmd(cmd.Stdout), nil
}
