// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import "encoding/xml"

// pomProject represents a parsed Maven pom.xml file.
type pomProject struct {
	XMLName      xml.Name        `xml:"project"`
	GroupId      string          `xml:"groupId"`
	ArtifactId   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Parent       *pomParent      `xml:"parent"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`

	// evidence is a list of file paths where the pom.xml was found.
	evidence []string `json:"-"`
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
