// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagelockjson

import (
	"encoding/json"
	"sort"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript"
)

// scopeOf maps an npm lock package entry to a languages package scope: a dev-only
// package reports PackageScopeDev, everything else (including devOptional, which
// can appear in the production tree) PackageScopeProd.
func scopeOf(pkg packageLockPackage) string {
	if pkg.Dev && !pkg.DevOptional {
		return languages.PackageScopeDev
	}
	return languages.PackageScopeProd
}

// dependsOnRefs resolves a package entry's `dependencies` (name→version-range)
// to the refs (purls) of the resolved packages, following npm's hoisting rules
// from fromPath. Returns sorted, deduplicated purls; skips deps that do not
// resolve to a `packages` entry. This yields the package→package edges the
// lockfile encodes (lockfileVersion 2+).
func dependsOnRefs(pkgs map[string]packageLockPackage, fromPath string, deps map[string]string) []string {
	if len(deps) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var refs []string
	for name := range deps {
		if ref := resolveDepPurl(pkgs, fromPath, name); ref != "" && !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	sort.Strings(refs)
	return refs
}

// resolveDepPurl finds the package entry that satisfies depName as required from
// fromPath, per npm resolution: try fromPath/node_modules/depName, then walk up
// the node_modules chain to the root. Returns the resolved package's purl, or ""
// when no entry is found.
func resolveDepPurl(pkgs map[string]packageLockPackage, fromPath, depName string) string {
	base := fromPath
	for {
		cand := "node_modules/" + depName
		if base != "" {
			cand = base + "/node_modules/" + depName
		}
		if entry, ok := pkgs[cand]; ok {
			// Build the ref from the resolved entry's path key, exactly as
			// Transitive() builds each package's own Purl, so an edge ref matches
			// its target node's Purl (a self-consistent graph).
			return javascript.NewPackageUrl(cand, entry.Version)
		}
		if base == "" {
			return ""
		}
		if idx := strings.LastIndex(base, "/node_modules/"); idx >= 0 {
			base = base[:idx]
		} else {
			base = ""
		}
	}
}

// packageLock is the struct to represent the package.lock file
// see https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
type packageLock struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	LockfileVersion int    `json:"lockfileVersion"`
	Requires        bool   `json:"requires"`
	// Packages maps package locations to an object containing the information about that package,
	// root project is typically listed with a key of ""
	Packages map[string]packageLockPackage `json:"packages"`
	// Dependencies contains legacy data for supporting versions of npm that use lockfileVersion: 1 or lower.
	// We can ignore that for lockfileVersion: 2+
	Dependencies map[string]packageLockDependency `jsonn:"dependencies"`

	// evidence is a list of file paths where the package-lock was found
	evidence []string `json:"-"`
}

type packageLockDependency struct {
	Version   string `json:"version"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
	Dev       bool   `json:"dev"`
}

type packageLockPackage struct {
	Name      string             `json:"name"`
	Version   string             `json:"version"`
	Resolved  string             `json:"resolved"`
	Integrity string             `json:"integrity"`
	License   packageLockLicense `json:"license"`
	// Dev marks a development-only dependency (npm lockfileVersion 2+): present
	// in the tree only for devDependencies / their closure, not the production
	// runtime. Also DevOptional (a package needed both as dev and optionally in
	// prod) — treated as prod, since it can be in the production tree.
	Dev          bool              `json:"dev"`
	DevOptional  bool              `json:"devOptional"`
	Dependencies map[string]string `json:"dependencies"`
}

type packageLockLicense []string

// UnmarshalJSON is a custom unmarshaler for the packageLockLicense type. It allows to handle the license field
// which could be either a string or an array.
func (l *packageLockLicense) UnmarshalJSON(data []byte) (err error) {

	var slice []string
	if err := json.Unmarshal(data, &slice); err == nil {
		*l = slice
		return nil
	}

	var single string
	if err = json.Unmarshal(data, &single); err == nil {
		*l = []string{single}
		return nil
	}

	// if it's neither a string nor an array, ignore it
	return nil
}

func packageLockPackageName(path string) string {
	return strings.TrimPrefix(path, "node_modules/")
}
