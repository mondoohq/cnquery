// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package selfupdate

import (
	"os"
	"syscall"
)

// ExecUpdatedBinary replaces the current process with the updated binary.
// On Unix systems, this uses syscall.Exec which replaces the current process entirely.
func ExecUpdatedBinary(binaryPath string, args []string) error {
	// Disable engine auto-update in the new process to prevent infinite update loops.
	// Provider auto-update (which reads MONDOO_AUTO_UPDATE via viper) is not affected.
	os.Setenv(EnvAutoUpdateEngine, "false")

	// Replace the current process with the new binary
	// syscall.Exec replaces the current process image with the new one
	return syscall.Exec(binaryPath, args, os.Environ())
}
