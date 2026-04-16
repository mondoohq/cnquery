// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cargotoml

import "github.com/BurntSushi/toml"

// cargoToml represents a parsed Cargo.toml file.
type cargoToml struct {
	Package           cargoPackageInfo          `toml:"package"`
	Dependencies      map[string]toml.Primitive `toml:"dependencies"`
	DevDependencies   map[string]toml.Primitive `toml:"dev-dependencies"`
	BuildDependencies map[string]toml.Primitive `toml:"build-dependencies"`

	// Resolved after TOML decoding (not populated by toml.Decode directly).
	resolvedDeps      []resolvedDep `toml:"-"`
	resolvedDevDeps   []resolvedDep `toml:"-"`
	resolvedBuildDeps []resolvedDep `toml:"-"`

	// evidence is a list of file paths where the Cargo.toml was found.
	evidence []string `toml:"-"`
}

// resolvedDep holds the resolved name and version of a dependency.
type resolvedDep struct {
	Name    string
	Version string
}

// cargoPackageInfo represents the [package] section.
type cargoPackageInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Edition string `toml:"edition"`
}

// cargoDep represents a dependency in table format.
type cargoDep struct {
	Version  string `toml:"version"`
	Git      string `toml:"git"`
	Path     string `toml:"path"`
	Optional bool   `toml:"optional"`
}
