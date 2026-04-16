// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packageslockjson

import (
	"encoding/json"
	"io"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/dotnet"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*packagesLock)(nil)
)

// Extractor parses NuGet packages.lock.json files.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "packageslockjson"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var lock packagesLock
	if err := json.NewDecoder(r).Decode(&lock); err != nil {
		return nil, err
	}

	if filename != "" {
		lock.evidence = append(lock.evidence, filename)
	}

	return &lock, nil
}

// allPackages returns a deduplicated list of packages across all target frameworks.
func (l *packagesLock) allPackages() map[string]packagesLockPackage {
	seen := make(map[string]packagesLockPackage)
	for _, framework := range l.Dependencies {
		for name, pkg := range framework {
			// First occurrence wins (frameworks typically have the same versions)
			if _, ok := seen[name]; !ok {
				seen[name] = pkg
			}
		}
	}
	return seen
}

// Root returns nil — packages.lock.json does not contain root project info.
func (l *packagesLock) Root() *languages.Package {
	return nil
}

// Direct returns packages with type "Direct".
func (l *packagesLock) Direct() languages.Packages {
	var direct languages.Packages
	for name, pkg := range l.allPackages() {
		if pkg.isDirect() {
			direct = append(direct, makePackage(name, pkg.Resolved, l.evidence))
		}
	}
	return direct
}

// Transitive returns all packages (both direct and transitive).
func (l *packagesLock) Transitive() languages.Packages {
	var all languages.Packages
	for name, pkg := range l.allPackages() {
		all = append(all, makePackage(name, pkg.Resolved, l.evidence))
	}
	return all
}

func makePackage(name, version string, evidence []string) *languages.Package {
	return &languages.Package{
		Name:         name,
		Version:      version,
		Purl:         dotnet.NewPackageUrl(name, version),
		Cpes:         dotnet.NewCpes(name, version),
		EvidenceList: dotnet.NewEvidenceList(evidence),
	}
}
