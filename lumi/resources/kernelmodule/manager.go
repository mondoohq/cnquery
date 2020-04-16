package kernelmodule

import (
	"github.com/pkg/errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type KernelModule struct {
	Name   string
	Size   string // int64
	UsedBy string // int
}

type OSKernelModuleManager interface {
	Name() string
	List() ([]*KernelModule, error)
}

func ResolveManager(motor *motor.Motor) (OSKernelModuleManager, error) {
	var kmm OSKernelModuleManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		kmm = &OSXKernelModuleManager{motor: motor}
	} else if platform.IsFamily("linux") {
		kmm = &LinuxKernelModuleManager{motor: motor}
	} else if platform.Name == "freebsd" {
		// NOTE: kldstat may work on other bsd linux
		kmm = &FreebsdKernelModuleManager{motor: motor}
	}

	if kmm == nil {
		return nil, errors.New("could not detect suitable kernel module manager for platform: " + platform.Name)
	}

	return kmm, nil
}

type LinuxKernelModuleManager struct {
	motor *motor.Motor
}

func (s *LinuxKernelModuleManager) Name() string {
	return "Linux Kernel Module Manager"
}

func (s *LinuxKernelModuleManager) List() ([]*KernelModule, error) {
	// TODO: use proc in future
	cmd, err := s.motor.Transport.RunCommand("lsmod")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseLsmod(cmd.Stdout), nil
}

type OSXKernelModuleManager struct {
	motor *motor.Motor
}

func (s *OSXKernelModuleManager) Name() string {
	return "macOS Kernel Module Manager"
}

func (s *OSXKernelModuleManager) List() ([]*KernelModule, error) {
	cmd, err := s.motor.Transport.RunCommand("kextstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKextstat(cmd.Stdout), nil
}

type FreebsdKernelModuleManager struct {
	motor *motor.Motor
}

func (s *FreebsdKernelModuleManager) Name() string {
	return "Freebsd Kernel Module Manager"
}

func (s *FreebsdKernelModuleManager) List() ([]*KernelModule, error) {
	cmd, err := s.motor.Transport.RunCommand("kldstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKldstat(cmd.Stdout), nil
}
