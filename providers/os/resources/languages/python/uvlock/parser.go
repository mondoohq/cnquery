// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package uvlock

// uvLock represents a parsed uv.lock file.
type uvLock struct {
	Version  int         `toml:"version"`
	Packages []uvPackage `toml:"package"`

	// evidence is a list of file paths where the uv.lock was found.
	evidence []string `toml:"-"`
}

// uvPackage represents a single [[package]] entry in uv.lock.
type uvPackage struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Source  uvSource `toml:"source"`
}

// uvSource describes where a package was sourced from.
type uvSource struct {
	Virtual  string `toml:"virtual"`
	Registry string `toml:"registry"`
	Git      string `toml:"git"`
}

// isRoot returns true if this package is the root project (virtual source).
func (p *uvPackage) isRoot() bool {
	return p.Source.Virtual == "."
}
