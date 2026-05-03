// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/sbom"
)

type PackageDetails struct {
	Name           string
	File           string
	License        string
	Author         string
	AuthorEmail    string
	Summary        string
	Version        string
	RequiresPython string
	ProjectUrls    map[string]string
	Dependencies   []string
	IsLeaf         bool
	Purl           string
	Cpes           []string
}

func NewPackageUrl(name string, version string) string {
	// ensure the name is according to the PURL spec
	// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#pypi
	name = strings.ToLower(name)
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
	cpeEntries, err := cpe.NewPackage2Cpe(vendor, name, version, "", "")
	if err != nil {
		log.Warn().Str("name", name).Str("version", version).Err(err).Msg("failed to create cpe for Python package")
	} else if len(cpeEntries) > 0 {
		cpes = append(cpes, cpeEntries...)
	}
	return cpes
}

// NewEvidenceList converts a list of file paths to evidence entries.
func NewEvidenceList(evidence []string) []*sbom.Evidence {
	evidenceList := make([]*sbom.Evidence, len(evidence))
	for i, e := range evidence {
		evidenceList[i] = NewEvidence(e)
	}
	return evidenceList
}

// NewEvidence creates a file evidence entry.
func NewEvidence(filepath string) *sbom.Evidence {
	return &sbom.Evidence{
		Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
		Value: filepath,
	}
}
