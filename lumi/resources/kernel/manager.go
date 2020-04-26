package kernel

import (
	"github.com/pkg/errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type KernelModule struct {
	Name   string
	Size   string // int64
	UsedBy string // int
}

type OSKernelManager interface {
	Name() string
	Parameters() (map[string]string, error)
	Modules() ([]*KernelModule, error)
}

func ResolveManager(motor *motor.Motor) (OSKernelManager, error) {
	var kmm OSKernelManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		kmm = &OSXKernelManager{motor: motor}
	} else if platform.IsFamily("linux") {
		kmm = &LinuxKernelManager{motor: motor}
	} else if platform.Name == "freebsd" {
		// NOTE: kldstat may work on other bsd linux
		kmm = &FreebsdKernelManager{motor: motor}
	}

	if kmm == nil {
		return nil, errors.New("could not detect suitable kernel module manager for platform: " + platform.Name)
	}

	return kmm, nil
}

type LinuxKernelManager struct {
	motor *motor.Motor
}

func (s *LinuxKernelManager) Name() string {
	return "Linux Kernel Module Manager"
}

func (s *LinuxKernelManager) Parameters() (map[string]string, error) {
	cmd, err := s.motor.Transport.RunCommand("/sbin/sysctl -a")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel parameters")
	}

	return ParseSysctl(cmd.Stdout, "=")
}

func (s *LinuxKernelManager) Modules() ([]*KernelModule, error) {
	// TODO: use proc in future
	cmd, err := s.motor.Transport.RunCommand("/sbin/lsmod")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseLsmod(cmd.Stdout), nil
}

type OSXKernelManager struct {
	motor *motor.Motor
}

func (s *OSXKernelManager) Name() string {
	return "macOS Kernel Manager"
}

func (s *OSXKernelManager) Parameters() (map[string]string, error) {
	cmd, err := s.motor.Transport.RunCommand("sysctl -a")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel parameters")
	}

	return ParseSysctl(cmd.Stdout, ":")
}

func (s *OSXKernelManager) Modules() ([]*KernelModule, error) {
	cmd, err := s.motor.Transport.RunCommand("kextstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKextstat(cmd.Stdout), nil
}

type FreebsdKernelManager struct {
	motor *motor.Motor
}

func (s *FreebsdKernelManager) Name() string {
	return "Freebsd Kernel Manager"
}

func (s *FreebsdKernelManager) Parameters() (map[string]string, error) {
	cmd, err := s.motor.Transport.RunCommand("sysctl -a")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel parameters")
	}

	return ParseSysctl(cmd.Stdout, ":")
}

func (s *FreebsdKernelManager) Modules() ([]*KernelModule, error) {
	cmd, err := s.motor.Transport.RunCommand("kldstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKldstat(cmd.Stdout), nil
}
