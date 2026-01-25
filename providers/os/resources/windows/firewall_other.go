// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/cnquery/v12/providers/os/connection/shared"

// GetNativeFirewallProfiles returns nil on non-Windows platforms to trigger PowerShell fallback
func GetNativeFirewallProfiles(conn shared.Connection) ([]WindowsFirewallProfile, error) {
	return nil, nil
}

// GetNativeFirewallRules returns nil on non-Windows platforms to trigger PowerShell fallback
func GetNativeFirewallRules(conn shared.Connection) ([]WindowsFirewallRule, error) {
	return nil, nil
}

// GetNativeFirewallSettings returns nil on non-Windows platforms to trigger PowerShell fallback
func GetNativeFirewallSettings(conn shared.Connection) (*WindowsFirewallSettings, error) {
	return nil, nil
}
