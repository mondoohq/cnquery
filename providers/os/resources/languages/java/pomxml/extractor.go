// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"encoding/xml"
	"io"

	"go.mondoo.com/mql/providers/os/resources/languages"
	"go.mondoo.com/mql/providers/os/resources/languages/java"
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
		if p.isTestOrProvided(dep) {
			continue
		}
		direct = append(direct, p.depToPackage(dep))
	}
	return direct
}

// Transitive returns all declared dependencies (direct only — pom.xml does not
// resolve transitives without downloading the full dependency tree).
func (p *pomProject) Transitive() languages.Packages {
	var all languages.Packages
	for _, dep := range p.Dependencies {
		all = append(all, p.depToPackage(dep))
	}
	return all
}

// depToPackage renders one <dependency> as a package, resolving the ${...}
// property references and the <dependencyManagement> version that decide what
// the dependency's identity actually is.
//
// Both matter beyond tidiness. A version left as the literal "${jackson.version}"
// produces a purl no advisory database and no package registry can match, so the
// dependency is silently exempt from vulnerability correlation and from licence
// lookup alike.
func (p *pomProject) depToPackage(dep pomDependency) *languages.Package {
	groupId := p.resolve(dep.GroupId)
	artifactId := p.resolve(dep.ArtifactId)

	version := p.resolve(dep.Version)
	if version == "" {
		// No <version> on the dependency: the project's <dependencyManagement>
		// is where it is declared, which is the standard way a multi-module
		// project states a version once.
		version = p.resolve(p.managedVersion(groupId, artifactId))
	}

	name := artifactId
	if groupId != "" {
		name = groupId + ":" + artifactId
	}

	return &languages.Package{
		Name:         name,
		Version:      version,
		Purl:         java.NewPackageUrl(groupId, artifactId, version),
		Cpes:         java.NewCpes(groupId, artifactId, version),
		EvidenceList: java.NewEvidenceList(p.evidence),
	}
}
