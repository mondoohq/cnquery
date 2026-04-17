// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package lockfile

// terraformLock represents a parsed .terraform.lock.hcl file.
type terraformLock struct {
	// Providers is the list of locked provider entries.
	Providers []providerEntry

	// evidence is a list of file paths where the lock file was found.
	evidence []string
}

// providerEntry represents a single provider block in the lock file.
type providerEntry struct {
	// Source is the full provider source address (e.g., "registry.terraform.io/hashicorp/aws").
	Source string
	// Version is the resolved provider version.
	Version string
}
