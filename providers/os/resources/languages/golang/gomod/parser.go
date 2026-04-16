// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gomod

// goMod represents a parsed go.mod file.
type goMod struct {
	// Module is the module path declared in the module directive.
	Module string
	// GoVersion is the Go version from the go directive.
	GoVersion string
	// Require is the list of required dependencies.
	Require []goModRequire
	// Replace maps original module paths to their replacements.
	Replace map[string]goModRequire

	// evidence is a list of file paths where the go.mod was found.
	evidence []string `json:"-"`
}

// goModRequire represents a single require directive entry.
type goModRequire struct {
	// Path is the module path (e.g., "github.com/pkg/errors").
	Path string
	// Version is the module version (e.g., "v0.9.1").
	Version string
	// Indirect indicates whether this is an indirect dependency.
	Indirect bool
}
