// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package services

import "errors"

// GetNativeWindowsServices is a stub for non-Windows platforms.
// This will never be called at runtime since the detection logic
// checks for local Windows connections first.
func GetNativeWindowsServices() ([]*Service, error) {
	return nil, errors.New("native Windows services not supported on non-Windows platforms")
}
