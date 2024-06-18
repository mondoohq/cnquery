// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package fs

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// handleFsError checks if the error is a permission denied or non-existent file error and returns nil in such cases. the bool
// indicates if the file should be skipped
func handleFsError(err error) (bool, error) {
	if err != nil {
		// Check for denied permissions and non-existent files. This can sometimes happen, especially for procfs
		// We don't want to error out in such cases. We can safely skip over the file.
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrInvalid) || errors.Is(err, unix.EINVAL) {
			return true, nil
		}
		return true, err
	}
	return false, nil
}
