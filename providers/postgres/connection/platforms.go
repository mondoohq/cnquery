// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Platforms is the static catalog of platforms this provider can emit.
var Platforms = []*plugin.PlatformInfo{
	{
		Name:    "postgres",
		Title:   "PostgreSQL Server",
		Family:  []string{"postgres"},
		Kind:    []string{"api"},
		Runtime: []string{"postgres"},
	},
	{
		Name:    "postgres-database",
		Title:   "PostgreSQL Database",
		Family:  []string{"postgres"},
		Kind:    []string{"api"},
		Runtime: []string{"postgres"},
	},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the catalog entry for the given platform name.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}
