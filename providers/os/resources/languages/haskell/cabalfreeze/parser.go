// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cabalfreeze

// cabalFreeze represents a parsed cabal.project.freeze file.
type cabalFreeze struct {
	// Packages is the list of pinned dependencies.
	Packages []cabalPackage

	// evidence is a list of file paths where the freeze file was found.
	evidence []string
}

// cabalPackage represents a single pinned dependency.
type cabalPackage struct {
	Name    string
	Version string
}
