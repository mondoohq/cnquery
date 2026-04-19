// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import (
	"encoding/json"
	"io"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/conan"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*conanLock)(nil)
)

// Extractor parses Conan lock files (v1 and v2).
type Extractor struct{}

func (e *Extractor) Name() string {
	return "conanlock"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var lock conanLock

	if filename != "" {
		lock.evidence = append(lock.evidence, filename)
	}

	if err := json.NewDecoder(r).Decode(&lock); err != nil {
		return nil, err
	}

	return &lock, nil
}

// Root returns nil for v2, or the root node (node "0") for v1.
func (l *conanLock) Root() *languages.Package {
	if l.GraphLock != nil {
		if node, ok := l.GraphLock.Nodes["0"]; ok && node.Path != "" {
			name, version := conan.ParseConanRef(node.Ref)
			if name != "" {
				return &languages.Package{
					Name:         name,
					Version:      version,
					Purl:         conan.NewPackageUrl(name, version),
					Cpes:         conan.NewCpes(name, version),
					EvidenceList: conan.NewEvidenceList(l.evidence),
				}
			}
		}
	}
	return nil
}

// Direct returns production dependencies.
// V2: "requires" list. V1: all non-root nodes.
func (l *conanLock) Direct() languages.Packages {
	if l.isV2() {
		return l.parseRefs(l.Requires)
	}
	// V1 doesn't distinguish direct from transitive
	return nil
}

// Transitive returns all dependencies (requires + build_requires for v2, all nodes for v1).
func (l *conanLock) Transitive() languages.Packages {
	if l.isV2() {
		var all languages.Packages
		all = append(all, l.parseRefs(l.Requires)...)
		all = append(all, l.parseRefs(l.BuildRequires)...)
		all = append(all, l.parseRefs(l.PythonRequires)...)
		return all
	}

	// V1: all non-root nodes
	if l.GraphLock == nil {
		return nil
	}

	var all languages.Packages
	for id, node := range l.GraphLock.Nodes {
		if id == "0" {
			continue // skip root
		}
		name, version := conan.ParseConanRef(node.Ref)
		if name == "" {
			continue
		}
		all = append(all, &languages.Package{
			Name:         name,
			Version:      version,
			Purl:         conan.NewPackageUrl(name, version),
			Cpes:         conan.NewCpes(name, version),
			EvidenceList: conan.NewEvidenceList(l.evidence),
		})
	}
	return all
}

func (l *conanLock) parseRefs(refs []string) languages.Packages {
	var packages languages.Packages
	for _, ref := range refs {
		name, version := conan.ParseConanRef(ref)
		if name == "" {
			continue
		}
		packages = append(packages, &languages.Package{
			Name:         name,
			Version:      version,
			Purl:         conan.NewPackageUrl(name, version),
			Cpes:         conan.NewCpes(name, version),
			EvidenceList: conan.NewEvidenceList(l.evidence),
		})
	}
	return packages
}
