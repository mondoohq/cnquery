// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package processes

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

var (
	modKernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snapshot  = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW           = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW            = modKernel32.NewProc("Process32NextW")
	procQueryFullProcessImageName = modKernel32.NewProc("QueryFullProcessImageNameW")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
)

// PROCESSENTRY32W structure for Windows process enumeration
// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/ns-tlhelp32-processentry32w
type processEntry32W struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

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
	command := p.ExePath
	if command == "" {
		command = p.ExeName
	}
	return &OSProcess{
		Pid:        int64(p.PID),
		Command:    command,
		Executable: p.ExeName,
		State:      p.State,
	}
}

// GetNativeProcessList retrieves the list of processes using native Windows API
func GetNativeProcessList() ([]*NativeWindowsProcess, error) {
	log.Debug().Msg("enumerating processes using native Windows API")

	// Create a snapshot of all processes
	handle, _, err := procCreateToolhelp32Snapshot.Call(
		uintptr(TH32CS_SNAPPROCESS),
		0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(windows.Handle(handle))

	var processes []*NativeWindowsProcess

	// Initialize the process entry structure
	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	// Get the first process
	ret, _, err := procProcess32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, fmt.Errorf("Process32First failed: %w", err)
	}

	for {
		exeName := windows.UTF16ToString(entry.ExeFile[:])

		proc := &NativeWindowsProcess{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Threads:   entry.Threads,
			ExeName:   exeName,
			State:     "running", // Processes in the snapshot are running
		}

		// Try to get the full executable path
		proc.ExePath = getProcessImagePath(entry.ProcessID)

		processes = append(processes, proc)

		// Reset entry size for next iteration
		entry.Size = uint32(unsafe.Sizeof(entry))

		// Get the next process
		ret, _, _ = procProcess32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	log.Debug().Int("count", len(processes)).Msg("found processes via native API")
	return processes, nil
}

// getProcessImagePath retrieves the full path of a process executable
func getProcessImagePath(pid uint32) string {
	// Skip system processes that can't be opened
	if pid == 0 || pid == 4 {
		return ""
	}

	// Try to open the process with query limited information
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		pid,
	)
	if err != nil {
		// This is expected for system processes and processes owned by other users
		return ""
	}
	defer windows.CloseHandle(handle)

	// Query the full process image name
	var buf [MAX_PATH * 2]uint16
	size := uint32(len(buf))

	ret, _, _ := procQueryFullProcessImageName.Call(
		uintptr(handle),
		0, // Win32 path format
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return ""
	}

	return windows.UTF16ToString(buf[:size])
}

// GetNativeProcess retrieves a specific process by PID using native Windows API
func GetNativeProcess(pid int64) (*NativeWindowsProcess, error) {
	if pid < 0 {
		return nil, errors.New("invalid PID")
	}

	processes, err := GetNativeProcessList()
	if err != nil {
		return nil, err
	}

	for _, proc := range processes {
		if int64(proc.PID) == pid {
			return proc, nil
		}
	}

	return nil, fmt.Errorf("process %d not found", pid)
}

// NativeProcessExists checks if a process with the given PID exists using native Windows API
func NativeProcessExists(pid int64) (bool, error) {
	if pid < 0 {
		return false, nil
	}

	// Try to open the process - this is faster than enumerating all processes
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		uint32(pid),
	)
	if err != nil {
		// Process doesn't exist or we don't have permission
		// Check via enumeration to distinguish between these cases
		processes, enumErr := GetNativeProcessList()
		if enumErr != nil {
			return false, enumErr
		}
		for _, proc := range processes {
			if int64(proc.PID) == pid {
				return true, nil
			}
		}
		return false, nil
	}
	windows.CloseHandle(handle)
	return true, nil
}
