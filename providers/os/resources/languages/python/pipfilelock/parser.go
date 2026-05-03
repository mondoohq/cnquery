// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pipfilelock

import "strings"

// pipfileLock represents a parsed Pipfile.lock file.
type pipfileLock struct {
	Default map[string]pipfilePackage `json:"default"`
	Develop map[string]pipfilePackage `json:"develop"`

	// evidence is a list of file paths where the Pipfile.lock was found.
	evidence []string `json:"-"`
}

// pipfilePackage represents a single dependency entry in Pipfile.lock.
type pipfilePackage struct {
	Version string   `json:"version"`
	Hashes  []string `json:"hashes"`
}

// cleanVersion strips the PEP 440 "==" prefix from a pinned version string.
func cleanVersion(version string) string {
	return strings.TrimPrefix(version, "==")
}
