// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gemfilelock

// gemfileLock represents a parsed Gemfile.lock file.
type gemfileLock struct {
	// Gems is the list of resolved gems from the GEM specs section.
	Gems []gemEntry
	// DirectDeps is the list of direct dependency names from the DEPENDENCIES section.
	DirectDeps map[string]bool
	// BundledWith is the Bundler version used to create this lock file.
	BundledWith string

	// evidence is a list of file paths where the Gemfile.lock was found.
	evidence []string
}

// gemEntry represents a single resolved gem in the GEM specs section.
type gemEntry struct {
	Name    string
	Version string
}
