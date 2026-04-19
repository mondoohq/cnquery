// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mixlock

// mixLock represents a parsed Elixir mix.lock file.
type mixLock struct {
	// Packages is the list of resolved Hex packages.
	Packages []mixPackage

	// evidence is a list of file paths where the lock file was found.
	evidence []string
}

// mixPackage represents a single package entry in mix.lock.
type mixPackage struct {
	Name    string
	Version string
}
