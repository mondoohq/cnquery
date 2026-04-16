// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package java

import (
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/sbom"
)

// NewPackageUrl creates a Maven package URL for a given groupId, artifactId, and version.
// See https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#maven
func NewPackageUrl(groupId, artifactId, version string) string {
	return packageurl.NewPackageURL(
		packageurl.TypeMaven,
		groupId,
		artifactId,
		version,
		nil,
		"").String()
}

// NewCpes creates CPE entries for a Maven package.
func NewCpes(groupId, artifactId, version string) []string {
	cpes := []string{}

	// Derive vendor from groupId: use last meaningful segment.
	// e.g., "org.apache.commons" -> "apache"
	// e.g., "com.google.guava" -> "google"
	vendor := vendorFromGroupId(groupId)

	cpeEntries, err := cpe.NewPackage2Cpe(vendor, artifactId, version, "", "")
	if err != nil {
		log.Warn().Str("groupId", groupId).Str("artifactId", artifactId).Str("version", version).Err(err).Msg("failed to create cpe for Java package")
	} else if len(cpeEntries) > 0 {
		cpes = append(cpes, cpeEntries...)
	}
	return cpes
}

// vendorFromGroupId extracts a vendor name from a Maven groupId.
// It skips common TLD prefixes (org, com, net, io) and returns the
// next segment as the vendor.
func vendorFromGroupId(groupId string) string {
	parts := strings.Split(groupId, ".")
	if len(parts) == 0 {
		return groupId
	}

	// Skip common TLD/domain prefixes
	skipPrefixes := map[string]bool{
		"org": true, "com": true, "net": true, "io": true,
		"de": true, "fr": true, "uk": true, "co": true,
	}

	for _, part := range parts {
		if !skipPrefixes[part] {
			return part
		}
	}

	// If all parts were skipped (unlikely), use the full groupId
	return groupId
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
