// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cargolock

import (
	"io"

	"github.com/BurntSushi/toml"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/rust"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*cargoLock)(nil)
)

// Extractor parses Cargo.lock files to extract Rust crate dependencies.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "cargolock"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var lock cargoLock
	if _, err := toml.NewDecoder(r).Decode(&lock); err != nil {
		return nil, err
	}

	if filename != "" {
		lock.evidence = append(lock.evidence, filename)
	}

	return &lock, nil
}

// Root returns the root project package (the entry without a source field).
func (l *cargoLock) Root() *languages.Package {
	for _, pkg := range l.Packages {
		if pkg.isRoot() {
			return &languages.Package{
				Name:         pkg.Name,
				Version:      pkg.Version,
				Purl:         rust.NewPackageUrl(pkg.Name, pkg.Version),
				Cpes:         rust.NewCpes(pkg.Name, pkg.Version),
				EvidenceList: rust.NewEvidenceList(l.evidence),
			}
		}
	}
	return nil
}

// Direct returns nil — Cargo.lock does not distinguish direct from transitive.
// Use Cargo.toml for direct dependency information.
func (l *cargoLock) Direct() languages.Packages {
	return nil
}

// Transitive returns all packages (excluding the root project).
func (l *cargoLock) Transitive() languages.Packages {
	var packages languages.Packages
	for _, pkg := range l.Packages {
		if pkg.isRoot() {
			continue
		}
		packages = append(packages, &languages.Package{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Purl:         rust.NewPackageUrl(pkg.Name, pkg.Version),
			Cpes:         rust.NewCpes(pkg.Name, pkg.Version),
			EvidenceList: rust.NewEvidenceList(l.evidence),
		})
	}
	return packages
}
