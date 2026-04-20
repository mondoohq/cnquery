// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package podfilelock

// podfileLock represents a parsed Podfile.lock file.
type podfileLock struct {
	// Pods is the list of resolved pods with versions.
	Pods []podEntry

	// evidence is a list of file paths where the Podfile.lock was found.
	evidence []string
}

// podEntry represents a single pod with its resolved version.
type podEntry struct {
	Name    string
	Version string
}
