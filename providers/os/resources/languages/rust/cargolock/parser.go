// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cargolock

// cargoLock represents a parsed Cargo.lock file.
type cargoLock struct {
	Version  int            `toml:"version"`
	Packages []cargoPackage `toml:"package"`

	// evidence is a list of file paths where the Cargo.lock was found.
	evidence []string `toml:"-"`
}

// cargoPackage represents a single [[package]] entry in Cargo.lock.
type cargoPackage struct {
	Name         string   `toml:"name"`
	Version      string   `toml:"version"`
	Source       string   `toml:"source"`
	Checksum     string   `toml:"checksum"`
	Dependencies []string `toml:"dependencies"`
}

// isRoot returns true if this package is the root project (no source field).
func (p *cargoPackage) isRoot() bool {
	return p.Source == ""
}
