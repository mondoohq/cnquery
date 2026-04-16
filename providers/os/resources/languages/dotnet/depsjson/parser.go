// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package depsjson

// depsJson represents a parsed *.deps.json file.
type depsJson struct {
	RuntimeTarget runtimeTarget          `json:"runtimeTarget"`
	Libraries     map[string]depsLibrary `json:"libraries"`

	// evidence is a list of file paths where the deps.json was found.
	evidence []string `json:"-"`
}

// runtimeTarget contains the target framework.
type runtimeTarget struct {
	Name string `json:"name"`
}

// depsLibrary represents a single library entry.
type depsLibrary struct {
	Type        string `json:"type"`
	Serviceable bool   `json:"serviceable"`
	Sha512      string `json:"sha512"`
}

// isPackage returns true if this library is a NuGet package (not a project reference).
func (l *depsLibrary) isPackage() bool {
	return l.Type == "package"
}
