// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package registry

import (
	"fmt"
	"os/exec"
)

func LoadRegistrySubkey(key, path string) error {
	return exec.Command("cmd", "/C", "reg", "load", fmt.Sprintf(`HKEY_LOCAL_MACHINE\%s`, key), path).Run()
}

func UnloadRegistrySubkey(key string) error {
	return exec.Command("cmd", "/C", "reg", "unload", fmt.Sprintf(`HKEY_LOCAL_MACHINE\%s`, key)).Run()
}
