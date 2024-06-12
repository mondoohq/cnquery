// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package registry

const (
	// According to https://learn.microsoft.com/en-gb/windows/win32/sysinfo/structure-of-the-registry
	// we have the following registry keys
	Software = "SOFTWARE"
	System   = "SYSTEM"
	Security = "SECURITY"
	Default  = "DEFAULT"
	Sam      = "SAM"

	SoftwareRegPath = "Windows\\System32\\config\\SOFTWARE"
	SystemRegPath   = "Windows\\System32\\config\\SYSTEM"
	SecurityRegPath = "Windows\\System32\\config\\SECURITY"
	DefaultRegPath  = "Windows\\System32\\config\\DEFAULT"
	SamRegPath      = "Windows\\System32\\config\\SAM"
)

var KnownRegistryFiles = map[string]string{
	Software: SoftwareRegPath,
	System:   SystemRegPath,
	Security: SecurityRegPath,
	Default:  DefaultRegPath,
	Sam:      SamRegPath,
}
