// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package yarnlock

import (
	"errors"
	"regexp"
	"strings"
)

type yarnLock map[string]yarnLockEntry

type yarnLockEntry struct {
	Version      string
	Resolved     string
	Dependencies map[string]string
}

var yarnPkgNameRe = regexp.MustCompile(`^(.*)@(.*)$`)

// parseYarnPackageName extracts the package name and version specifier from
// a yarn.lock map key. Keys may list multiple specifiers separated by commas
// (e.g. "has@^1.0.1, has@^1.0.3"); only the first is used.
//
// Returns (packageName, versionSpecifier, nil) on success.
// Returns ("", "", error) when the key cannot be parsed — for example
// non-package entries like "__metadata" in yarn berry (v2+) lockfiles.
func parseYarnPackageName(name string) (string, string, error) {
	pkgNames := strings.Split(name, ",")

	if len(pkgNames) == 0 {
		return "", "", errors.New("cannot parse yarn package name: " + name)
	}

	m := yarnPkgNameRe.FindStringSubmatch(strings.TrimSpace(pkgNames[0]))
	if len(m) < 3 {
		return "", "", errors.New("cannot parse yarn package name: " + name)
	}
	return m[1], m[2], nil
}
