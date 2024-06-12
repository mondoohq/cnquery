// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package registry

import (
	"syscall"
	"unsafe"
)

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	// note: we're using the W (RegLoadKeyW and NOT RegLoadKeyA) versions of these functions to work with UTF16 strings
	regLoadKey   = advapi32.NewProc("RegLoadKeyW")
	regUnloadKey = advapi32.NewProc("RegUnLoadKeyW")
)

func LoadRegistrySubkey(key, path string) error {
	keyPtr, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return err
	}
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	ret, _, err := regLoadKey.Call(syscall.HKEY_LOCAL_MACHINE, uintptr(unsafe.Pointer(keyPtr)), uintptr(unsafe.Pointer(pathPtr)))
	// the Microsoft docs indicate that the return value is 0 on success
	if ret != 0 {
		return err
	}
	return nil
}

func UnloadRegistrySubkey(key string) error {
	keyPtr, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return err
	}

	ret, _, err := regUnloadKey.Call(syscall.HKEY_LOCAL_MACHINE, uintptr(unsafe.Pointer(keyPtr)))
	// the Microsoft docs indicate that the return value is 0 on success
	if ret != 0 {
		return err
	}
	return nil
}
