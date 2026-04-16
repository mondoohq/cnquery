// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package installedjson

// installedJson represents a parsed vendor/composer/installed.json file.
type installedJson struct {
	Packages []installedPackage `json:"packages"`

	// evidence is a list of file paths where the installed.json was found.
	evidence []string `json:"-"`
}

// installedPackage represents a single installed package entry.
type installedPackage struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	License     []string `json:"license"`
	Description string   `json:"description"`
}
