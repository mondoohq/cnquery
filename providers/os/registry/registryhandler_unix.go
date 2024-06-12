// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package registry

import (
	"errors"
)

func LoadRegistrySubkey(key, path string) error {
	return errors.New("LoadRegistrySubkey is not supported on non-windows platforms")
}

func UnloadRegistrySubkey(key string) error {
	return errors.New("UnloadRegistrySubkey is not supported on non-windows platforms")
}
