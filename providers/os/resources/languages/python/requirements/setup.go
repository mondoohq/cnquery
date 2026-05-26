// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package requirements

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// pinnedQuoted matches quoted pinned dependencies in setup.py, e.g.:
//
//	'pathlib3==2.2.0;python_version<"3.6"'
//	"mypy==v0.770",
var pinnedQuoted = regexp.MustCompile(`['"]([a-zA-Z0-9][\w.\-]*)\s*==\s*([\w.\-]+)`)

// pinnedUnquoted matches unquoted pinned dependencies, e.g.:
//
//	mypy == v0.770
var pinnedUnquoted = regexp.MustCompile(`^\s*([a-zA-Z0-9][\w.\-]*)\s*==\s*([\w.\-]+)`)

// ParseSetupPy scans a setup.py (or setup.cfg) file for pinned dependencies
// (name==version). It extracts both quoted strings inside install_requires
// lists and bare unquoted assignments.
func ParseSetupPy(r io.Reader) ([]Requirement, error) {
	scanner := bufio.NewScanner(r)
	seen := map[string]bool{}
	var reqs []Requirement

	for scanner.Scan() {
		line := scanner.Text()

		// Quoted dependencies (can appear multiple times per line)
		for _, m := range pinnedQuoted.FindAllStringSubmatch(line, -1) {
			name, version := m[1], m[2]
			if hasTemplate(name) || hasTemplate(version) {
				continue
			}
			key := strings.ToLower(name) + "==" + version
			if seen[key] {
				continue
			}
			seen[key] = true
			reqs = append(reqs, Requirement{Name: name, Version: version})
		}

		// Unquoted dependency (one per line)
		if m := pinnedUnquoted.FindStringSubmatch(line); m != nil {
			name, version := m[1], m[2]
			if hasTemplate(name) || hasTemplate(version) {
				continue
			}
			key := strings.ToLower(name) + "==" + version
			if seen[key] {
				continue
			}
			seen[key] = true
			reqs = append(reqs, Requirement{Name: name, Version: version})
		}
	}

	return reqs, scanner.Err()
}

func hasTemplate(s string) bool {
	return strings.ContainsAny(s, "%{}$")
}
