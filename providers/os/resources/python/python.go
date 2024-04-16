// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"strings"
)

func NewPackageUrl(name string, version string) string {
	// ensure the name is according to the PURL spec
	// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#pypi
	name = strings.ReplaceAll(name, "_", "-")

	return packageurl.NewPackageURL(
		packageurl.TypePyPi,
		"",
		name,
		version,
		nil,
		"").String()
}

func NewCpes(name string, version string) []string {
	cpes := []string{}
	// what we see in the cpe dictionary is that the vendor is the name of the package itself + "_project"
	vendor := name + "_project"
	cpeEntry, err := cpe.NewPackage2Cpe(vendor, name, version, "", "")
	if err == nil && cpeEntry != "" {
		cpes = append(cpes, cpeEntry)
	}
	return cpes
}
