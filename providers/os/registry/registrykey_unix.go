// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package registry

import "errors"

// non-windows stubs
func GetNativeRegistryKeyItems(path string) ([]RegistryKeyItem, error) {
	return nil, errors.New("native registry key items not supported on non-windows platforms")
}

func GetNativeRegistryKeyChildren(path string) ([]RegistryKeyChild, error) {
	return nil, errors.New("native registry key children not supported on non-windows platforms")
}

func GetNativeRegistryKeyItem(path, key string) (RegistryKeyItem, error) {
	return RegistryKeyItem{}, errors.New("native registry key item not supported on non-windows platforms")
}
