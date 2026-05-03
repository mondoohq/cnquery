// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import (
	"strconv"
	"strings"
)

// conanLock represents a parsed conan.lock file.
type conanLock struct {
	Version        string         `json:"version"`
	GraphLock      conanGraphLock `json:"graph_lock"`
	Requires       []string       `json:"requires"`
	BuildRequires  []string       `json:"build_requires"`
	PythonRequires []string       `json:"python_requires"`

	// evidence is a list of file paths where the conan.lock was found.
	evidence []string `json:"-"`
}

// conanGraphLock represents the v1 graph_lock format.
type conanGraphLock struct {
	Nodes map[string]conanGraphNode `json:"nodes"`
}

// conanGraphNode represents a single node in the v1 graph lock.
type conanGraphNode struct {
	Ref  string `json:"ref"`
	Pref string `json:"pref"`
	Path string `json:"path"`
}

// isV2 returns true if this is a v2 format lockfile (version >= 0.5).
func (l *conanLock) isV2() bool {
	v, err := strconv.ParseFloat(l.Version, 64)
	if err != nil {
		return false
	}
	return v >= 0.5
}

// conanReference holds the parsed components of a Conan reference string.
type conanReference struct {
	Name    string
	Version string
}

// parseConanReference parses a Conan reference string in the format:
//
//	name/version[@user[/channel]][#revision]
//
// Returns the name and version components.
func parseConanReference(ref string) (conanReference, bool) {
	if ref == "" {
		return conanReference{}, false
	}

	// Strip everything after # (recipe revision).
	if idx := strings.IndexByte(ref, '#'); idx >= 0 {
		ref = ref[:idx]
	}

	// Strip everything after @ (user/channel).
	if idx := strings.IndexByte(ref, '@'); idx >= 0 {
		ref = ref[:idx]
	}

	// Split name/version.
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return conanReference{}, false
	}

	return conanReference{
		Name:    parts[0],
		Version: parts[1],
	}, true
}
