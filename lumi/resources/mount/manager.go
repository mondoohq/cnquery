package mount

import (
	"github.com/pkg/errors"
	motor "go.mondoo.io/mondoo/motor/motoros"
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

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			mm = &LinuxMountManager{motor: motor}
			break
		} else if platform.Family[i] == "unix" {
			mm = &UnixMountManager{motor: motor}
			break
		}
	}

	if mm == nil {
		return nil, errors.New("could not detect suitable mount manager for platform: " + platform.Name)
	}

	return mm, nil
}

type LinuxMountManager struct {
	motor *motor.Motor
}

func (s *LinuxMountManager) Name() string {
	return "Linux Mount Manager"
}

func (s *LinuxMountManager) List() ([]MountPoint, error) {
	// TODO: not working via docker yet
	// // try /proc
	// f, err := s.motor.Transport.File("/proc/mount")
	// if err == nil {
	// 	defer f.Close()
	// 	return ParseLinuxProcMount(f), nil
	// }

	// fallback to mount cmd
	cmd, err := s.motor.Transport.RunCommand("mount")
	if err != nil {
		return nil, errors.Wrap(err, "could not read mounts")
	}

	return ParseLinuxMountCmd(cmd.Stdout), nil
}

type UnixMountManager struct {
	motor *motor.Motor
}

func (s *UnixMountManager) Name() string {
	return "Unix Mount Manager"
}

func (s *UnixMountManager) List() ([]MountPoint, error) {
	cmd, err := s.motor.Transport.RunCommand("mount")
	if err != nil {
		return nil, errors.Wrap(err, "could not read package list")
	}

	return ParseUnixMountCmd(cmd.Stdout), nil
}
