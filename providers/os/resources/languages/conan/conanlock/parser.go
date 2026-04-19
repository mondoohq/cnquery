// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

// conanLock represents a parsed conan.lock file (supports both v1 and v2).
type conanLock struct {
	// V2 fields
	Version        string   `json:"version"`
	Requires       []string `json:"requires"`
	BuildRequires  []string `json:"build_requires"`
	PythonRequires []string `json:"python_requires"`

	// V1 fields
	GraphLock *graphLock `json:"graph_lock"`

	// evidence is a list of file paths where the lock file was found.
	evidence []string `json:"-"`
}

// graphLock is the v1 lock file structure.
type graphLock struct {
	Nodes map[string]graphNode `json:"nodes"`
}

// graphNode is a single node in the v1 dependency graph.
type graphNode struct {
	Ref      string   `json:"ref"`
	Path     string   `json:"path"`
	Requires []string `json:"requires"`
	Context  string   `json:"context"`
}

// isV2 returns true if this is a Conan 2.x lock file.
// V1 always has graph_lock, so its absence identifies v2.
func (l *conanLock) isV2() bool {
	return l.GraphLock == nil
}
