// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mount

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
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

func ResolveManager(conn shared.Connection) (OperatingSystemMountManager, error) {
	var mm OperatingSystemMountManager

	pf := conn.Asset().Platform
	if pf == nil {
		return nil, errors.New("missing platform information")
	}

	if pf.IsFamily("linux") {
		mm = &LinuxMountManager{conn: conn}
	} else if pf.IsFamily("unix") {
		mm = &UnixMountManager{conn: conn}
	}

	if mm == nil {
		return nil, errors.New("could not detect suitable mount manager for platform: " + pf.Name)
	}

	return mm, nil
}

type LinuxMountManager struct {
	conn shared.Connection
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

	if s.conn.Capabilities().Has(shared.Capability_RunCommand) {
		cmd, err := s.conn.RunCommand("mount")
		if err != nil {
			return nil, errors.Wrap(err, "could not read mounts")
		}
		return ParseLinuxMountCmd(cmd.Stdout), nil
	} else if s.conn.Capabilities().Has(shared.Capability_File) {
		return mountsFromFSLinux(s.conn.FileSystem())
	}

	return nil, errors.New("mount not supported for provided transport")
}

type UnixMountManager struct {
	conn shared.Connection
}

func (s *UnixMountManager) Name() string {
	return "Unix Mount Manager"
}

func (s *UnixMountManager) List() ([]MountPoint, error) {
	cmd, err := s.conn.RunCommand("mount")
	if err != nil {
		return nil, errors.Wrap(err, "could not read package list")
	}

	return ParseUnixMountCmd(cmd.Stdout), nil
}
