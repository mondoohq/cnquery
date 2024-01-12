// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import "go.mondoo.com/cnquery/v10/providers/os/connection/shared"

type ScratchPkgManager struct {
	conn shared.Connection
}

func (dpm *ScratchPkgManager) Name() string {
	return "Scratch Package Manager"
}

func (dpm *ScratchPkgManager) Format() string {
	return "scratch"
}

func (dpm *ScratchPkgManager) List() ([]Package, error) {
	return []Package{}, nil
}

func (dpm *ScratchPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
