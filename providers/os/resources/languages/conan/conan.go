// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conan

import (
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/sbom"
)

// NewPackageUrl creates a Conan package URL.
// See https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#conan
func NewPackageUrl(name string, version string) string {
	return packageurl.NewPackageURL(
		"conan",
		"",
		name,
		version,
		nil,
		"").String()
}

// NewCpes creates CPE entries for a Conan package.
func NewCpes(name string, version string) []string {
	cpes := []string{}
	cpeEntries, err := cpe.NewPackage2Cpe(name, name, version, "", "")
	if err != nil {
		log.Warn().Str("name", name).Str("version", version).Err(err).Msg("failed to create cpe for Conan package")
	} else if len(cpeEntries) > 0 {
		cpes = append(cpes, cpeEntries...)
	}
	return cpes
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

// ParseConanRef parses a Conan reference like "name/version#revision" into name and version.
func ParseConanRef(ref string) (name, version string) {
	// Strip revision hash: "name/version#revision" -> "name/version"
	if idx := strings.Index(ref, "#"); idx != -1 {
		ref = ref[:idx]
	}

	// Split on "/": "name/version" -> name, version
	if n, v, ok := strings.Cut(ref, "/"); ok {
		return n, v
	}

	return ref, ""
}
