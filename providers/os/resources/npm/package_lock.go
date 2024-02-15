// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"io"

	"encoding/json"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
)

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

// PackageLockParser is the parser for the package.lock file npm format.
// see https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
type PackageLockParser struct{}

func (p *PackageLockParser) Parse(r io.Reader) ([]*mvd.Package, error) {
	var packageJsonLock packageLock
	err := json.NewDecoder(r).Decode(&packageJsonLock)
	if err != nil {
		return nil, err
	}

	entries := []*mvd.Package{}

	// add own package
	entries = append(entries, &mvd.Package{
		Name:      packageJsonLock.Name,
		Version:   packageJsonLock.Version,
		Format:    "npm",
		Namespace: "nodejs",
	})

	// add all dependencies
	for k, v := range packageJsonLock.Dependencies {
		entries = append(entries, &mvd.Package{
			Name:      k,
			Version:   v.Version,
			Format:    "npm",
			Namespace: "nodejs",
		})
	}

	return entries, nil
}
