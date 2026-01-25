// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package users

import "errors"

// GetNativeUsers is a stub for non-Windows platforms.
// Native Windows user enumeration is only available on Windows.
func GetNativeUsers() ([]*User, error) {
	return nil, errors.New("native user enumeration not supported on non-Windows platforms")
}
