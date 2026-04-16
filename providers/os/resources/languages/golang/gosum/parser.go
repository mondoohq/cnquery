// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gosum

// goSum represents a parsed go.sum file.
type goSum struct {
	// Entries maps "module@version" to its hash entries.
	Entries []goSumEntry

	// evidence is a list of file paths where the go.sum was found.
	evidence []string `json:"-"`
}

// goSumEntry represents a single entry in a go.sum file.
type goSumEntry struct {
	// Path is the module path.
	Path string
	// Version is the module version.
	Version string
	// Hash is the content hash (h1:...).
	Hash string
}
