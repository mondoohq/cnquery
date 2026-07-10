// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagelockjson

import (
	"encoding/json"
	"io"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*packageLock)(nil)
)

// Extractor is the parser for the package.lock file npm format.
// see https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
type Extractor struct {
}

func (p *Extractor) Name() string {
	return "packagelockjson"
}

func (p *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
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

func (p *packageLock) Root() *languages.Package {
	root := &languages.Package{
		Name:         p.Name,
		Version:      p.Version,
		Purl:         javascript.NewPackageUrl(p.Name, p.Version),
		Cpes:         javascript.NewCpes(p.Name, p.Version),
		EvidenceList: javascript.NewEvidenceList(p.evidence),
	}
	return root
}

func (p *packageLock) Direct() languages.Packages {
	// search for root package, read the packages field

	// at this point we only support lockfileVersion: 2 with direct dependencies
	if p.Packages == nil {
		return nil
	}

	rootPkg, ok := p.Packages[""]
	if !ok {
		return nil
	}

	filteredList := []*languages.Package{}
	for name := range rootPkg.Dependencies {
		// The root's declared dependencies are keyed in `packages` by their
		// install path, node_modules/<name> (npm hoists direct deps to the root
		// node_modules), not by bare name. Look them up there; keying by bare
		// name matched nothing for lockfileVersion 2+, so Direct() returned an
		// empty set. Build Name/Purl/Cpes from the path key exactly as
		// Transitive() does, so a package's Direct and Transitive representations
		// (and their refs) are identical.
		path := "node_modules/" + name
		pkg, ok := p.Packages[path]
		if !ok {
			continue
		}

		filteredList = append(filteredList, &languages.Package{
			Name:         packageLockPackageName(path),
			Version:      pkg.Version,
			Purl:         javascript.NewPackageUrl(path, pkg.Version),
			Cpes:         javascript.NewCpes(path, pkg.Version),
			EvidenceList: javascript.NewEvidenceList(p.evidence),
			DependsOn:    dependsOnRefs(p.Packages, path, pkg.Dependencies),
		})
	}

	return filteredList
}

func (p *packageLock) Transitive() languages.Packages {
	var transitive languages.Packages
	if p.Packages != nil {
		for k, v := range p.Packages {
			name := k
			// skip root package since we have that already
			if name == "" {
				name = v.Name
			}

			transitive = append(transitive, &languages.Package{
				Name:         packageLockPackageName(name),
				Version:      v.Version,
				Purl:         javascript.NewPackageUrl(k, v.Version),
				Cpes:         javascript.NewCpes(k, v.Version),
				EvidenceList: javascript.NewEvidenceList(p.evidence),
				DependsOn:    dependsOnRefs(p.Packages, k, v.Dependencies),
			})
		}
	} else if p.Dependencies != nil {
		for k, v := range p.Dependencies {
			transitive = append(transitive, &languages.Package{
				Name:         k,
				Version:      v.Version,
				Purl:         javascript.NewPackageUrl(k, v.Version),
				Cpes:         javascript.NewCpes(k, v.Version),
				EvidenceList: javascript.NewEvidenceList(p.evidence),
			})
		}
	}
	return transitive
}
