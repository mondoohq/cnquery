// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package languages

import (
	"cmp"
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
	Root() *Package
	// Direct dependencies of the root package.
	Direct() Packages
	// Transitive dependencies of the root package.
	Transitive() Packages
}

// Package represents a software package.
type Package struct {
	// The name of the package.
	Name string `json:"name,omitempty"`
	// The version of the package.
	Version string `json:"version,omitempty"`
	// The architecture of the package
	Architecture string `json:"architecture,omitempty"`
	// The Common Platform Enumeration (CPE) name
	Cpes []string `json:"cpes,omitempty"`
	// The Package URL (pURL), a standardized format for defining and locating
	// software package metadata.
	Purl string `json:"purl,omitempty"`
	// location on disk
	// Deprecated: use evidence instead
	Location string `json:"location,omitempty"`
	// 'type' indicates the type of package, such as a rpm, dpkg, or gem.
	Type string `json:"type,omitempty"`
	// description of the package
	Description string `json:"description,omitempty"`
	// 'evidence_list' is a collection of evidence that supports the presence of
	// the package in the asset. This evidence could include eg. file paths
	EvidenceList []*sbom.Evidence `json:"evidence_list,omitempty"`
	// Package Origin (e.g. other package name, or source of the package)
	Origin string `json:"origin,omitempty"`
	// Package Vendor/Publisher
	Vendor string `json:"vendor,omitempty"`
}

// SortFn is a helper function for slices.SortFunc to sort a slice of Package
// by name and version. Use it like this: slices.SortFunc(packages, sbom.SortFn)
func SortFn(a, b *Package) int {
	if n := cmp.Compare(a.Name, b.Name); n != 0 {
		return n
	}
	// if names are equal, order by version
	return cmp.Compare(a.Version, b.Version)
}

type Packages []*Package

// Find a package by name.
func (p Packages) Find(name string) *Package {
	for _, pkg := range p {
		if pkg.Name == name {
			return pkg
		}
	}
	return nil
}
