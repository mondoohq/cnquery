// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"encoding/xml"
	"io"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/java"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*pomProject)(nil)
)

// Extractor parses Maven pom.xml files to extract project dependencies.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "pomxml"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	project, err := parsePomXml(r)
	if err != nil {
		return nil, err
	}

	if filename != "" {
		project.evidence = append(project.evidence, filename)
	}

	return project, nil
}

// parsePomXml reads and parses a Maven pom.xml file.
func parsePomXml(r io.Reader) (*pomProject, error) {
	var project pomProject
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&project); err != nil {
		return nil, err
	}
	return &project, nil
}

// Root returns the project itself as the root package.
func (p *pomProject) Root() *languages.Package {
	groupId := p.effectiveGroupId()
	version := p.effectiveVersion()

	if groupId == "" && p.ArtifactId == "" {
		return nil
	}

	name := p.ArtifactId
	if groupId != "" {
		name = groupId + ":" + p.ArtifactId
	}

	return &languages.Package{
		Name:         name,
		Version:      version,
		Purl:         java.NewPackageUrl(groupId, p.ArtifactId, version),
		Cpes:         java.NewCpes(groupId, p.ArtifactId, version),
		EvidenceList: java.NewEvidenceList(p.evidence),
	}
}

// Direct returns production dependencies (scope != test, scope != provided).
func (p *pomProject) Direct() languages.Packages {
	var direct languages.Packages
	for _, dep := range p.Dependencies {
		if dep.isTestOrProvided() {
			continue
		}
		direct = append(direct, depToPackage(dep, p.evidence))
	}
	return direct
}

// Transitive returns all declared dependencies (direct only — pom.xml does not
// resolve transitives without downloading the full dependency tree).
func (p *pomProject) Transitive() languages.Packages {
	var all languages.Packages
	for _, dep := range p.Dependencies {
		all = append(all, depToPackage(dep, p.evidence))
	}
	return all
}

func depToPackage(dep pomDependency, evidence []string) *languages.Package {
	name := dep.ArtifactId
	if dep.GroupId != "" {
		name = dep.GroupId + ":" + dep.ArtifactId
	}

	return &languages.Package{
		Name:         name,
		Version:      dep.Version,
		Purl:         java.NewPackageUrl(dep.GroupId, dep.ArtifactId, dep.Version),
		Cpes:         java.NewCpes(dep.GroupId, dep.ArtifactId, dep.Version),
		EvidenceList: java.NewEvidenceList(evidence),
	}
}
