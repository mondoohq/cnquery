// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import "strconv"

// conanLock represents a parsed conan.lock file.
type conanLock struct {
	Version   string         `json:"version"`
	GraphLock conanGraphLock `json:"graph_lock"`
	Requires  []string       `json:"requires"`
	// BuildRequires, PythonRequires and ConfigRequires are all build-time
	// tooling rather than content of the produced artifact. ConfigRequires
	// arrived with Conan 2's `conan config install-pkg`; before it was read
	// here, a lockfile's config packages were dropped entirely — not scoped
	// out, but absent, so `--include-dev` did not bring them back either.
	BuildRequires  []string `json:"build_requires"`
	PythonRequires []string `json:"python_requires"`
	ConfigRequires []string `json:"config_requires"`

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
	// Context is "build" for a node in the build context (a tool_requires and
	// its closure) and "host" — or absent — otherwise. It is how a v1 lockfile
	// says what v2 says with a separate build_requires list.
	Context string `json:"context"`
}

// isV2 returns true if this is a v2 format lockfile (version >= 0.5).
func (l *conanLock) isV2() bool {
	v, err := strconv.ParseFloat(l.Version, 64)
	if err != nil {
		return false
	}
	return v >= 0.5
}
