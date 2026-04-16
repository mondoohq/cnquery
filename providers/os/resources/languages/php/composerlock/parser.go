// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package composerlock

// composerLock represents a parsed composer.lock file.
type composerLock struct {
	ContentHash string            `json:"content-hash"`
	Packages    []composerPackage `json:"packages"`
	PackagesDev []composerPackage `json:"packages-dev"`

	// evidence is a list of file paths where the lock file was found.
	evidence []string `json:"-"`
}

// composerPackage represents a single package entry in composer.lock.
type composerPackage struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	License     []string `json:"license"`
	Description string   `json:"description"`
}
