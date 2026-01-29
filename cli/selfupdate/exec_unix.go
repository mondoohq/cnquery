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
	// Disable auto-update for the new process to prevent infinite loops
	os.Setenv(EnvAutoUpdate, "false")

	// Replace the current process with the new binary
	// syscall.Exec replaces the current process image with the new one
	return syscall.Exec(binaryPath, args, os.Environ())
}
