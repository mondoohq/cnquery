// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pdmlock

// pdmLock represents a parsed pdm.lock file.
type pdmLock struct {
	Packages []pdmPackage `toml:"package"`

	// evidence is a list of file paths where the pdm.lock was found.
	evidence []string `toml:"-"`
}

// pdmPackage represents a single [[package]] entry in pdm.lock.
type pdmPackage struct {
	Name     string   `toml:"name"`
	Version  string   `toml:"version"`
	Groups   []string `toml:"groups"`
	Revision string   `toml:"revision"`
}
