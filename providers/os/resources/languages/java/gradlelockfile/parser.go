// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gradlelockfile

// gradleLockfile represents a parsed gradle.lockfile.
type gradleLockfile struct {
	// Entries is the list of locked dependencies.
	Entries []gradleLockEntry

	// evidence is a list of file paths where the lockfile was found.
	evidence []string `json:"-"`
}

// gradleLockEntry represents a single entry in a gradle.lockfile.
type gradleLockEntry struct {
	// GroupId is the Maven group ID (e.g., "org.apache.commons").
	GroupId string
	// ArtifactId is the Maven artifact ID (e.g., "commons-lang3").
	ArtifactId string
	// Version is the resolved version (e.g., "3.12.0").
	Version string
	// Configurations is the list of Gradle configurations this dependency belongs to.
	Configurations []string
	// IsTest indicates whether this dependency is only used in test configurations.
	IsTest bool
}
