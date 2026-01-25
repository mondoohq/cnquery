// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package processes

import "errors"

var errNotWindows = errors.New("native Windows process API is not available on this platform")

// NativeWindowsProcess represents a process retrieved via native Windows API
type NativeWindowsProcess struct {
	PID        uint32
	ParentPID  uint32
	Threads    uint32
	ExeName    string
	ExePath    string
	State      string
}

// ToOSProcess converts a NativeWindowsProcess to the generic OSProcess format
func (p *NativeWindowsProcess) ToOSProcess() *OSProcess {
	return nil
}

// GetNativeProcessList retrieves the list of processes using native Windows API
// This is a stub for non-Windows platforms
func GetNativeProcessList() ([]*NativeWindowsProcess, error) {
	return nil, errNotWindows
}

// GetNativeProcess retrieves a specific process by PID using native Windows API
// This is a stub for non-Windows platforms
func GetNativeProcess(pid int64) (*NativeWindowsProcess, error) {
	return nil, errNotWindows
}

// NativeProcessExists checks if a process with the given PID exists using native Windows API
// This is a stub for non-Windows platforms
func NativeProcessExists(pid int64) (bool, error) {
	return false, errNotWindows
}
