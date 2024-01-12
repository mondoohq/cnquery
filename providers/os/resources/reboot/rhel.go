// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reboot

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v10/providers/core/resources/versions/rpm"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/packages"
)

// RpmNewestKernel works on all machines running rpm
type RpmNewestKernel struct {
	conn shared.Connection
}

func (s *RpmNewestKernel) Name() string {
	return "RPM Latest Kernel"
}

func (s *RpmNewestKernel) RebootPending() (bool, error) {
	// if it is a static asset, no reboot is pending
	if !s.conn.Capabilities().Has(shared.Capability_RunCommand) {
		return false, nil
	}

	// get installed kernel version
	installedKernelCmd, err := s.conn.RunCommand("rpm -q kernel --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\n'")
	if err != nil {
		return false, err
	}

	var pf *inventory.Platform
	if s.conn.Asset() != nil {
		pf = s.conn.Asset().Platform
	}

	pkgs := packages.ParseRpmPackages(pf, installedKernelCmd.Stdout)
	// this case is valid in container
	if len(pkgs) == 0 {
		return false, nil
	}

	// check running kernel version
	unamerCmd, err := s.conn.RunCommand("uname -r")
	if err != nil {
		return false, err
	}

	unameR, err := io.ReadAll(unamerCmd.Stdout)
	if err != nil {
		return false, err
	}

	// check if any kernel is newer
	kernelVersion := strings.TrimSpace(string(unameR))

	var parser rpm.Parser

	for i := range pkgs {
		cmp, err := parser.Compare(pkgs[i].Version, kernelVersion)
		if err != nil {
			return false, err
		}
		if cmp >= 1 {
			return true, nil
		}
	}
	return false, nil
}
