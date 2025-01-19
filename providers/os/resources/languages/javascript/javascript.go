// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package javascript

import (
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/sbom"
)

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
		cleanVersion(version),
		nil,
		"").String()
}

func NewCpes(name string, version string) []string {
	cpes := []string{}
	cpeEntries, err := cpe.NewPackage2Cpe(name, name, cleanVersion(version), "", "")
	// we only add the cpe if it could be created
	// if the cpe could not be created, we log the error and continue to ensure the package is still added to the list
	if err != nil {
		log.Warn().Str("name", name).Str("version", version).Err(err).Msg("failed to create cpe")
	} else if len(cpeEntries) > 0 {
		cpes = append(cpes, cpeEntries...)
	}
	return cpes
}

func cleanVersion(version string) string {
	v := strings.ReplaceAll(version, "^", "")
	v = strings.ReplaceAll(v, "~", "")
	v = strings.ReplaceAll(v, ">", "")
	v = strings.ReplaceAll(v, "<", "")
	v = strings.ReplaceAll(v, "=", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func NewEvidenceList(evidence []string) []*sbom.Evidence {
	evidenceList := make([]*sbom.Evidence, len(evidence))
	for i, e := range evidence {
		evidenceList[i] = NewEvidence(e)
	}
	return evidenceList
}

func NewEvidence(filepath string) *sbom.Evidence {
	return &sbom.Evidence{
		Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
		Value: filepath,
	}
}
