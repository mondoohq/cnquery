// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package csproj

import "encoding/xml"

// project represents a parsed .csproj / .fsproj / .vbproj file.
type project struct {
	XMLName    xml.Name    `xml:"Project"`
	ItemGroups []itemGroup `xml:"ItemGroup"`

	// evidence is a list of file paths where the project file was found.
	evidence []string `xml:"-"`
}

// itemGroup contains PackageReference elements.
type itemGroup struct {
	PackageReferences []packageReference `xml:"PackageReference"`
}

// packageReference represents a single <PackageReference> element.
type packageReference struct {
	Include       string `xml:"Include,attr"`
	Version       string `xml:"Version,attr"`
	PrivateAssets string `xml:"PrivateAssets,attr"`
}

// isDev returns true if this is a development-only package (PrivateAssets="all").
func (p *packageReference) isDev() bool {
	return p.PrivateAssets == "all"
}
