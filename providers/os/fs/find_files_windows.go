// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package fs

import (
	"errors"
	"os"
)

// handleFsError checks if the error is a permission denied or non-existent file error and returns nil in such cases.
func handleFsError(err error) error {
	if err != nil {
		// Check for denied permissions and non-existent files. This can sometimes happen, especially for procfs
		// We don't want to error out in such cases. We can safely skip over the file.
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrInvalid) {
			return nil
		}
		return err
	}
	return nil
}
