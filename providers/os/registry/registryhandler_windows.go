// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package registry

import (
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
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
	_, _, err = regLoadKey.Call(syscall.HKEY_LOCAL_MACHINE, uintptr(unsafe.Pointer(keyPtr)), uintptr(unsafe.Pointer(pathPtr)))
	// the Microsoft docs indicate that the return value is 0 on success
	if syserr, ok := err.(syscall.Errno); ok && syserr != 0 {
		log.Debug().Err(syserr).Msg("could not load registry subkey")
		return err
	}
	return nil
}

func UnloadRegistrySubkey(key string) error {
	keyPtr, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return err
	}

	_, _, err = regUnloadKey.Call(syscall.HKEY_LOCAL_MACHINE, uintptr(unsafe.Pointer(keyPtr)))
	// the Microsoft docs indicate that the return value is 0 on success
	if syserr, ok := err.(syscall.Errno); ok && syserr != 0 {
		log.Debug().Err(syserr).Msg("could not unload registry subkey")
		return err
	}
	return nil
}
