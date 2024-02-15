// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"io"

	"encoding/json"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
)

type PackageJsonLockEntry struct {
	Version string `json:"version"`
	Dev     bool   `json:"dev"`
}

// PackageJsonLock is the struct to represent the package.lock file
type PackageJsonLock struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Dependencies map[string]PackageJsonLockEntry `jsonn:"dependencies"`
}

type PackageLockParser struct{}

func (p *PackageLockParser) Parse(r io.Reader) ([]*mvd.Package, error) {
	var packageJsonLock PackageJsonLock
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
