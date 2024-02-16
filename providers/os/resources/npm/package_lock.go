// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"io"
	"strings"

	"encoding/json"
)

var (
	_ Parser = (*PackageLockParser)(nil)
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
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Resolved     string             `json:"resolved"`
	Integrity    string             `json:"integrity"`
	License      packageLockLicense `json:"license"`
	Dependencies map[string]string  `json:"dependencies"`
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
type PackageLockParser struct {
}

func (p *PackageLockParser) Parse(r io.Reader, filename string) (NpmPackageInfo, error) {
	var packageJsonLock packageLock
	err := json.NewDecoder(r).Decode(&packageJsonLock)
	if err != nil {
		return nil, err
	}

	if filename != "" {
		packageJsonLock.evidence = append(packageJsonLock.evidence, filename)
	}

	return &packageJsonLock, nil
}

func (p *packageLock) Root() *Package {
	root := &Package{
		Name:              p.Name,
		Version:           p.Version,
		Purl:              NewPackageUrl(p.Name, p.Version),
		Cpes:              NewCpes(p.Name, p.Version),
		EvidenceLocations: p.evidence,
	}
	return root
}

func (p *packageLock) Direct() []*Package {
	// search for root package, read the packages field

	// at this point we only support lockfileVersion: 2 with direct dependencies
	if p.Packages == nil {
		return nil
	}

	rootPkg, ok := p.Packages[""]
	if !ok {
		return nil
	}

	filteredList := []*Package{}
	for k := range rootPkg.Dependencies {
		pkg, ok := p.Packages[k]
		if !ok {
			continue
		}

		filteredList = append(filteredList, &Package{
			Name:              packageLockPackageName(k),
			Version:           pkg.Version,
			Purl:              NewPackageUrl(k, pkg.Version),
			Cpes:              NewCpes(k, pkg.Version),
			EvidenceLocations: p.evidence,
		})
	}

	return filteredList
}

func (p *packageLock) Transitive() []*Package {
	transitive := []*Package{}
	if p.Packages != nil {
		for k, v := range p.Packages {
			name := k
			// skip root package since we have that already
			if name == "" {
				name = v.Name
			}

			transitive = append(transitive, &Package{
				Name:              packageLockPackageName(name),
				Version:           v.Version,
				Purl:              NewPackageUrl(k, v.Version),
				Cpes:              NewCpes(k, v.Version),
				EvidenceLocations: p.evidence,
			})
		}
	} else if p.Dependencies != nil {
		for k, v := range p.Dependencies {
			transitive = append(transitive, &Package{
				Name:              k,
				Version:           v.Version,
				Purl:              NewPackageUrl(k, v.Version),
				Cpes:              NewCpes(k, v.Version),
				EvidenceLocations: p.evidence,
			})
		}
	}
	return transitive
}

func packageLockPackageName(path string) string {
	return strings.TrimPrefix(path, "node_modules/")
}
