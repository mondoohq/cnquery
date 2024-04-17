// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"io"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const sysctlPath = "/proc/sys/"

type KernelInfo struct {
	Version   string            `json:"version"`
	Path      string            `json:"path"`
	Device    string            `json:"device"`
	Arguments map[string]string `json:"args"`
}

type KernelModule struct {
	Name   string
	Size   string // int64
	UsedBy string // int
}

type OSKernelManager interface {
	Name() string
	Parameters() (map[string]string, error)
	Modules() ([]*KernelModule, error)
	Info() (KernelInfo, error)
}

func ResolveManager(conn shared.Connection) (OSKernelManager, error) {
	var kmm OSKernelManager

	platform := conn.Asset().Platform

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		kmm = &OSXKernelManager{conn: conn}
	} else if platform.IsFamily("linux") {
		kmm = &LinuxKernelManager{conn: conn}
	} else if platform.Name == "freebsd" {
		// NOTE: kldstat may work on other bsd linux
		kmm = &FreebsdKernelManager{conn: conn}
	}

	if kmm == nil {
		return nil, errors.New("could not detect suitable kernel module manager for platform: " + platform.Name)
	}

	return kmm, nil
}

type LinuxKernelManager struct {
	conn shared.Connection
}

func (s *LinuxKernelManager) Name() string {
	return "Linux Kernel Module Manager"
}

func (s *LinuxKernelManager) Info() (KernelInfo, error) {
	res := KernelInfo{}

	cmdlineRaw, err := s.conn.FileSystem().Open("/proc/cmdline")
	if err != nil {
		return res, err
	}
	defer cmdlineRaw.Close()

	args, err := ParseLinuxKernelArguments(cmdlineRaw)
	if err != nil {
		return res, err
	}
	res.Path = args.Path
	res.Device = args.Device
	res.Arguments = args.Arguments

	versionRaw, err := s.conn.FileSystem().Open("/proc/version")
	if err != nil {
		return res, err
	}
	defer versionRaw.Close()

	version, err := ParseLinuxKernelVersion(versionRaw)
	if err != nil {
		return res, err
	}

	res.Version = version

	return res, nil
}

func (s *LinuxKernelManager) Parameters() (map[string]string, error) {
	if s.conn.Capabilities().Has(shared.Capability_RunCommand) {
		cmd, err := s.conn.RunCommand("/sbin/sysctl -a")
		// in case of err, the command is not there and we fallback to /proc/sys walking
		if err == nil && cmd.ExitStatus == 0 {
			log.Debug().Msg("using sysctl to read kernel parameters")
			return ParseSysctl(cmd.Stdout, "=")
		}
	}

	log.Debug().Msg("using /proc/sys walking to read kernel parameters")
	fs := s.conn.FileSystem()
	fsUtil := afero.Afero{Fs: fs}
	kernelParameters := make(map[string]string)
	err := fsUtil.Walk(sysctlPath, func(path string, f os.FileInfo, err error) error {
		if f != nil && !f.IsDir() {
			stat, err := s.conn.FileSystem().Stat(path)
			if err != nil {
				log.Error().Err(err)
				return nil
			}
			details := shared.FileModeDetails{
				FileMode: stat.Mode(),
			}
			if !details.UserReadable() {
				return nil
			}

			f, err := s.conn.FileSystem().Open(path)
			if err != nil {
				log.Error().Err(err)
				return err
			}

			content, err := io.ReadAll(f)
			if err != nil {
				log.Error().Err(err).Msg("cannot read content")
				return nil
			}
			// remove leading sysctl path
			k := strings.Replace(path, sysctlPath, "", -1)
			k = strings.Replace(k, "/", ".", -1)
			kernelParameters[k] = strings.TrimSpace(string(content))
		}
		return nil
	})

	return kernelParameters, err
}

func (s *LinuxKernelManager) Modules() ([]*KernelModule, error) {
	// TODO: use proc in future
	cmd, err := s.conn.RunCommand("/sbin/lsmod")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseLsmod(cmd.Stdout), nil
}

type OSXKernelManager struct {
	conn shared.Connection
}

func (s *OSXKernelManager) Name() string {
	return "macOS Kernel Manager"
}

func (s *OSXKernelManager) Info() (KernelInfo, error) {
	cmd, err := s.conn.RunCommand("uname -r")
	if err != nil {
		return KernelInfo{}, errors.Wrap(err, "could not read kernel parameters")
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return KernelInfo{}, errors.Wrap(err, "could not read kernel parameters")
	}

	return KernelInfo{
		Version: strings.TrimSpace(string(data)),
	}, nil
}

func (s *OSXKernelManager) Parameters() (map[string]string, error) {
	cmd, err := s.conn.RunCommand("sysctl -a")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel parameters")
	}

	return ParseSysctl(cmd.Stdout, ":")
}

func (s *OSXKernelManager) Modules() ([]*KernelModule, error) {
	cmd, err := s.conn.RunCommand("kextstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKextstat(cmd.Stdout), nil
}

type FreebsdKernelManager struct {
	conn shared.Connection
}

func (s *FreebsdKernelManager) Name() string {
	return "Freebsd Kernel Manager"
}

func (s *FreebsdKernelManager) Info() (KernelInfo, error) {
	return KernelInfo{}, nil
}

func (s *FreebsdKernelManager) Parameters() (map[string]string, error) {
	cmd, err := s.conn.RunCommand("sysctl -a")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel parameters")
	}

	return ParseSysctl(cmd.Stdout, ":")
}

func (s *FreebsdKernelManager) Modules() ([]*KernelModule, error) {
	cmd, err := s.conn.RunCommand("kldstat")
	if err != nil {
		return nil, errors.Wrap(err, "could not read kernel modules")
	}

	return ParseKldstat(cmd.Stdout), nil
}
