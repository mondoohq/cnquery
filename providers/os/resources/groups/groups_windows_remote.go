// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package groups

// List returns local groups on Windows via PowerShell.
// This is used when running on non-Windows systems (remote execution via SSH/WinRM).
func (s *WindowsGroupManager) List() ([]*Group, error) {
	return s.listViaPowershell()
}
