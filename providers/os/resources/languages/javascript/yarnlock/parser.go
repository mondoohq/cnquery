// Copyright (c) Mondoo, Inc.
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

func parseYarnPackageName(name string) (string, string, error) {
	// a yarn package line may include may items
	pkgNames := strings.Split(name, ",")

	if len(pkgNames) == 0 {
		// something wrong
		return "", "", errors.New("cannot parse yarn package name")
	}

	parse := regexp.MustCompile(`^(.*)@(.*)$`)
	m := parse.FindStringSubmatch(strings.TrimSpace(pkgNames[0]))
	return m[1], m[2], nil
}
