// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package rebarlock

// rebarLock represents a parsed Erlang rebar.lock file.
type rebarLock struct {
	// Packages is the list of resolved Hex packages.
	Packages []rebarPackage

	// evidence is a list of file paths where the lock file was found.
	evidence []string
}

// rebarPackage represents a single package entry in rebar.lock.
type rebarPackage struct {
	Name    string
	Version string
}
