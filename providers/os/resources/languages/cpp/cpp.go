// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cpp

import (
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/mql/v13/sbom"
)

// NewPackageUrl creates a Conan package URL for a given package name and version.
// See https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#conan
func NewPackageUrl(name string, version string) string {
	return packageurl.NewPackageURL(
		packageurl.TypeConan,
		"",
		name,
		version,
		nil,
		"").String()
}

// NewEvidenceList converts a list of file paths to evidence entries.
func NewEvidenceList(evidence []string) []*sbom.Evidence {
	evidenceList := make([]*sbom.Evidence, len(evidence))
	for i, e := range evidence {
		evidenceList[i] = &sbom.Evidence{
			Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
			Value: e,
		}
	}
	return evidenceList
}
