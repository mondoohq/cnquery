// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packageresolved

// packageResolved represents a parsed Package.resolved file (v1 or v2).
type packageResolved struct {
	// Version is the Package.resolved format version (1 or 2).
	Version int `json:"version"`
	// Pins is the list of resolved dependencies (v2 format).
	Pins []pin `json:"pins"`
	// Object wraps pins in v1 format.
	Object *pinObject `json:"object"`

	// evidence is a list of file paths where the Package.resolved was found.
	evidence []string `json:"-"`
}

// pinObject is the v1 wrapper around pins.
type pinObject struct {
	Pins []pinV1 `json:"pins"`
}

// pin represents a single dependency in v2 format.
type pin struct {
	Identity string   `json:"identity"`
	Kind     string   `json:"kind"`
	Location string   `json:"location"`
	State    pinState `json:"state"`
}

// pinV1 represents a single dependency in v1 format.
type pinV1 struct {
	Package       string   `json:"package"`
	RepositoryURL string   `json:"repositoryURL"`
	State         pinState `json:"state"`
}

// pinState contains the resolved version and revision.
type pinState struct {
	Revision string `json:"revision"`
	Version  string `json:"version"`
}
