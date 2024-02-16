// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v10/providers/os/resources/cpe"
	"io"
	"strings"
)

type Parser interface {
	Parse(r io.Reader) (*Package, []*Package, error)
}

type Package struct {
	Name        string
	File        string
	License     string
	Description string
	Version     string
	Purl        string
	Cpes        []string
}

// NewPackageUrl creates a npm package url for a given package name and version
// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#npm
func NewPackageUrl(name string, version string) string {
	namespace := ""
	// ensure the name is according to the PURL spec
	name = strings.ReplaceAll(name, "_", "-")

	components := strings.Split(name, "/")
	if len(components) > 1 {
		namespace = components[0]
		name = components[1]
	}

	return packageurl.NewPackageURL(
		packageurl.TypeNPM,
		namespace,
		name,
		version,
		nil,
		"").String()
}

func NewCpes(name string, version string) []string {
	cpes := []string{}
	cpeEntry, err := cpe.NewPackage2Cpe(name, name, version, "", "")
	if err == nil && cpeEntry != "" {
		cpes = append(cpes, cpeEntry)
	}
	return cpes
}
