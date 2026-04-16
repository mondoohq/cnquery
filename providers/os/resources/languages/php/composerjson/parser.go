// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package composerjson

// composerJson represents a parsed composer.json file.
type composerJson struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Require     map[string]string `json:"require"`
	RequireDev  map[string]string `json:"require-dev"`

	// evidence is a list of file paths where the composer.json was found.
	evidence []string `json:"-"`
}
