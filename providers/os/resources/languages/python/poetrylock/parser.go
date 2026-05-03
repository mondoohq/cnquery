// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package poetrylock

// poetryLock represents a parsed poetry.lock file.
type poetryLock struct {
	Packages []poetryPackage `toml:"package"`

	// evidence is a list of file paths where the poetry.lock was found.
	evidence []string `toml:"-"`
}

// poetryPackage represents a single [[package]] entry in poetry.lock.
type poetryPackage struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
	Optional    bool   `toml:"optional"`
}
