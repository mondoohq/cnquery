// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package uvlock

import (
	"io"

	"github.com/BurntSushi/toml"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/python"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*uvLock)(nil)
)

// Extractor parses uv.lock files to extract Python package dependencies.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "uvlock"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var lock uvLock
	if _, err := toml.NewDecoder(r).Decode(&lock); err != nil {
		return nil, err
	}

	if filename != "" {
		lock.evidence = append(lock.evidence, filename)
	}

	return &lock, nil
}

// Root returns the root project package (the entry with virtual source ".").
func (l *uvLock) Root() *languages.Package {
	for _, pkg := range l.Packages {
		if pkg.isRoot() {
			return &languages.Package{
				Name:         pkg.Name,
				Version:      pkg.Version,
				Purl:         python.NewPackageUrl(pkg.Name, pkg.Version),
				Cpes:         python.NewCpes(pkg.Name, pkg.Version),
				EvidenceList: python.NewEvidenceList(l.evidence),
			}
		}
	}
	return nil
}

// Direct returns nil — uv.lock does not distinguish direct from transitive.
func (l *uvLock) Direct() languages.Packages {
	return nil
}

// Transitive returns all packages (excluding the root project).
func (l *uvLock) Transitive() languages.Packages {
	var packages languages.Packages
	for _, pkg := range l.Packages {
		if pkg.isRoot() {
			continue
		}
		packages = append(packages, &languages.Package{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Purl:         python.NewPackageUrl(pkg.Name, pkg.Version),
			Cpes:         python.NewCpes(pkg.Name, pkg.Version),
			EvidenceList: python.NewEvidenceList(l.evidence),
		})
	}
	return packages
}
