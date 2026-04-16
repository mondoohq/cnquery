// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagesconfig

import "encoding/xml"

// packagesConfig represents a parsed packages.config XML file.
type packagesConfig struct {
	XMLName  xml.Name        `xml:"packages"`
	Packages []configPackage `xml:"package"`

	// evidence is a list of file paths where the packages.config was found.
	evidence []string `xml:"-"`
}

// configPackage represents a single <package> element.
type configPackage struct {
	Id                    string `xml:"id,attr"`
	Version               string `xml:"version,attr"`
	TargetFramework       string `xml:"targetFramework,attr"`
	DevelopmentDependency string `xml:"developmentDependency,attr"`
}

// isDev returns true if this is a development-only dependency.
func (p *configPackage) isDev() bool {
	return p.DevelopmentDependency == "true"
}
