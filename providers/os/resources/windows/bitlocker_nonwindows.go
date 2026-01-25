// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package windows

import (
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// GetBitLockerVolumes retrieves BitLocker volume status using PowerShell.
// On non-Windows platforms, this always uses PowerShell as native WMI API
// is not available.
func GetBitLockerVolumes(conn shared.Connection) ([]bitlockerVolumeStatus, error) {
	return getPowershellBitLockerVolumes(conn)
}
