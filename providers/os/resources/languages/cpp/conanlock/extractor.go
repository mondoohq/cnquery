// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import (
	"encoding/json"
	"io"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/cpp"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*conanLock)(nil)
)

// Extractor parses conan.lock files to extract C/C++ package dependencies.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "conanlock"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var lock conanLock
	if err := json.NewDecoder(r).Decode(&lock); err != nil {
		return nil, err
	}

	if filename != "" {
		lock.evidence = append(lock.evidence, filename)
	}

	return &lock, nil
}

// Root returns nil — conan.lock does not contain a root project entry.
func (l *conanLock) Root() *languages.Package {
	return nil
}

// Direct returns nil — conan.lock does not distinguish direct from transitive.
func (l *conanLock) Direct() languages.Packages {
	return nil
}

// Transitive returns all packages listed in the lockfile.
func (l *conanLock) Transitive() languages.Packages {
	if l.isV2() {
		return l.parseV2()
	}
	return l.parseV1()
}

// parseV1 extracts packages from the v1 graph_lock format.
func (l *conanLock) parseV1() languages.Packages {
	var packages languages.Packages

	for _, node := range l.GraphLock.Nodes {
		// Skip local path dependencies.
		if node.Path != "" {
			continue
		}

		// Use pref (v0.3) or ref (v0.4+).
		ref := node.Ref
		if node.Pref != "" {
			ref = node.Pref
		}

		parsed, ok := parseConanReference(ref)
		if !ok {
			log.Warn().Str("ref", ref).Msg("cannot parse conan reference")
			continue
		}

		packages = append(packages, &languages.Package{
			Name:         parsed.Name,
			Version:      parsed.Version,
			Purl:         cpp.NewPackageUrl(parsed.Name, parsed.Version),
			EvidenceList: cpp.NewEvidenceList(l.evidence),
		})
	}

	return packages
}

// parseV2 extracts packages from the v2 requires/build_requires format.
func (l *conanLock) parseV2() languages.Packages {
	var packages languages.Packages

	allRefs := make([]string, 0, len(l.Requires)+len(l.BuildRequires)+len(l.PythonRequires))
	allRefs = append(allRefs, l.Requires...)
	allRefs = append(allRefs, l.BuildRequires...)
	allRefs = append(allRefs, l.PythonRequires...)

	for _, ref := range allRefs {
		parsed, ok := parseConanReference(ref)
		if !ok {
			log.Warn().Str("ref", ref).Msg("cannot parse conan reference")
			continue
		}

		packages = append(packages, &languages.Package{
			Name:         parsed.Name,
			Version:      parsed.Version,
			Purl:         cpp.NewPackageUrl(parsed.Name, parsed.Version),
			EvidenceList: cpp.NewEvidenceList(l.evidence),
		})
	}

	return packages
}
