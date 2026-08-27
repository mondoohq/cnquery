// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"encoding/xml"
	"strings"
)

// pomProject represents a parsed Maven pom.xml file.
type pomProject struct {
	XMLName    xml.Name   `xml:"project"`
	GroupId    string     `xml:"groupId"`
	ArtifactId string     `xml:"artifactId"`
	Version    string     `xml:"version"`
	Parent     *pomParent `xml:"parent"`
	// Properties are the user-defined ${...} substitutions a POM declares. A
	// version written as ${jackson.version} is extremely common — it is how a
	// project keeps a family of artifacts on one version — and without the bag
	// the dependency's version is the literal property reference, which matches
	// no package and no advisory.
	Properties pomProperties `xml:"properties"`
	// DependencyManagement holds versions declared once for the whole project
	// (or imported from a BOM). A <dependency> under <dependencies> may then
	// omit its <version> entirely, and the version stated here is the one that
	// applies.
	DependencyManagement struct {
		Dependencies []pomDependency `xml:"dependencies>dependency"`
	} `xml:"dependencyManagement"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`

	// evidence is a list of file paths where the pom.xml was found.
	evidence []string `json:"-"`
}

// pomProperties decodes <properties> as arbitrary child elements, which is what
// it is: every child element is a user-named property.
type pomProperties map[string]string

func (p *pomProperties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if *p == nil {
		*p = map[string]string{}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			// io.EOF on a truncated document: keep what was read rather than
			// discarding every property because the file ended early.
			return nil //nolint:nilerr
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var v string
			if err := d.DecodeElement(&v, &t); err != nil {
				return err
			}
			(*p)[t.Name.Local] = strings.TrimSpace(v)
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// maxPropertyDepth bounds property resolution. A property may refer to another
// property, and a POM may declare a cycle; the depth cap terminates both.
const maxPropertyDepth = 10

// resolve substitutes ${...} references in s using the project's properties and
// Maven's built-in project coordinates.
//
// An unresolvable reference is returned UNCHANGED, deliberately. A version this
// parser cannot resolve is not a version, and substituting a guess would report
// a dependency at some other version's identity — matching that version's
// advisories and that version's license. Leaving the reference intact keeps the
// gap visible to whatever reads it.
func (p *pomProject) resolve(s string) string {
	for depth := 0; depth < maxPropertyDepth; depth++ {
		if !strings.Contains(s, "${") {
			return s
		}
		next := p.expandOnce(s)
		if next == s {
			// Nothing more can be substituted; the rest stays as written.
			return s
		}
		s = next
	}
	return s
}

// expandOnce replaces every ${...} reference it can resolve, once.
func (p *pomProject) expandOnce(s string) string {
	var b strings.Builder
	for {
		start := strings.Index(s, "${")
		if start < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := strings.Index(s[start:], "}")
		if end < 0 {
			b.WriteString(s)
			return b.String()
		}
		end += start
		name := s[start+2 : end]
		b.WriteString(s[:start])
		if v, ok := p.property(name); ok {
			b.WriteString(v)
		} else {
			b.WriteString(s[start : end+1])
		}
		s = s[end+1:]
	}
}

// property resolves one property name: Maven's built-in project coordinates
// first, since a POM cannot redefine them, then the <properties> bag.
func (p *pomProject) property(name string) (string, bool) {
	switch name {
	case "project.version", "pom.version", "version":
		if v := p.effectiveVersion(); v != "" {
			return v, true
		}
		return "", false
	case "project.groupId", "pom.groupId", "groupId":
		if v := p.effectiveGroupId(); v != "" {
			return v, true
		}
		return "", false
	case "project.artifactId", "pom.artifactId", "artifactId":
		if p.ArtifactId != "" {
			return p.ArtifactId, true
		}
		return "", false
	}
	v, ok := p.Properties[name]
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// managedVersion returns the version <dependencyManagement> declares for a
// dependency, which is what applies when the <dependency> states none.
func (p *pomProject) managedVersion(groupId, artifactId string) string {
	for _, m := range p.DependencyManagement.Dependencies {
		if m.GroupId == groupId && m.ArtifactId == artifactId {
			return m.Version
		}
	}
	return ""
}

// pomParent represents the parent POM reference.
type pomParent struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// pomDependency represents a single <dependency> in a pom.xml.
type pomDependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}

// effectiveGroupId returns the project's groupId, inheriting from parent if not set.
func (p *pomProject) effectiveGroupId() string {
	if p.GroupId != "" {
		return p.GroupId
	}
	if p.Parent != nil {
		return p.Parent.GroupId
	}
	return ""
}

// effectiveVersion returns the project's version, inheriting from parent if not set.
func (p *pomProject) effectiveVersion() string {
	if p.Version != "" {
		return p.Version
	}
	if p.Parent != nil {
		return p.Parent.Version
	}
	return ""
}

// isTestOrProvided returns true if the dependency scope indicates a non-production dependency.
func (d *pomDependency) isTestOrProvided() bool {
	return d.Scope == "test" || d.Scope == "provided"
}
