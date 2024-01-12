// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reboot

import "go.mondoo.com/cnquery/v10/providers/os/connection/shared"

const LinuxRebootFile = "/var/run/reboot-required"

// DebianReboot works on Debian and Ubuntu
type DebianReboot struct {
	conn shared.Connection
}

func (s *DebianReboot) Name() string {
	return "Linux Reboot"
}

func (s *DebianReboot) RebootPending() (bool, error) {
	// try to stat the file
	_, err := s.conn.FileSystem().Stat(LinuxRebootFile)
	if err != nil {
		return false, nil
	}
	return true, nil
}
