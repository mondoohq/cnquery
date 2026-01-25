// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package windows

import "errors"

// GetNativeAuditpol is a stub for non-Windows platforms
func GetNativeAuditpol() ([]AuditpolEntry, error) {
	return nil, errors.New("native auditpol not supported on non-windows platforms")
}

// NativeAuditpolSupported returns false on non-Windows platforms
func NativeAuditpolSupported() bool {
	return false
}
