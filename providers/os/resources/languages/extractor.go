// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package languages

import (
	"io"

	"go.mondoo.com/cnquery/v11/sbom"
)

// Extractor is the common interface for all language specific bom extractors.
type Extractor interface {
	// Name of the extractor.
	Name() string
	// Parse the bom from the given reader.
	Parse(r io.Reader, filename string) (Bom, error)
}

// Bom represents a bill of materials for a specific language.
type Bom interface {
	// Root package of the bom.
	Root() *sbom.Package
	// Direct dependencies of the root package.
	Direct() Packages
	// Transitive dependencies of the root package.
	Transitive() Packages
}

type Packages []*sbom.Package

// Find a package by name.
func (p Packages) Find(name string) *sbom.Package {
	for _, pkg := range p {
		if pkg.Name == name {
			return pkg
		}
	}
	return nil
}
