// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package stacklock

// stackLock represents a parsed stack.yaml.lock file.
type stackLock struct {
	Packages []stackPackage `yaml:"packages"`

	// evidence is a list of file paths where the lock file was found.
	evidence []string `yaml:"-"`
}

// stackPackage represents a single package entry in stack.yaml.lock.
type stackPackage struct {
	Completed stackCompleted `yaml:"completed"`
	Original  stackOriginal  `yaml:"original"`
}

// stackCompleted contains the resolved package reference.
type stackCompleted struct {
	Hackage string `yaml:"hackage"`
}

// stackOriginal contains the original package reference.
type stackOriginal struct {
	Hackage string `yaml:"hackage"`
}
