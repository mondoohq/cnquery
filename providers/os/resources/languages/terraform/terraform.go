// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/cpe"
	"go.mondoo.com/mql/v13/sbom"
)

// defaultRegistry is stripped from provider source addresses.
const defaultRegistry = "registry.terraform.io/"

// NewPackageUrl creates a Terraform provider package URL.
func NewPackageUrl(namespace, providerType, version string) string {
	return packageurl.NewPackageURL(
		"terraform",
		namespace,
		providerType,
		version,
		nil,
		"").String()
}

// NewCpes creates CPE entries for a Terraform provider.
func NewCpes(namespace, providerType, version string) []string {
	cpes := []string{}
	cpeEntries, err := cpe.NewPackage2Cpe(namespace, providerType, version, "", "")
	if err != nil {
		log.Warn().Str("namespace", namespace).Str("type", providerType).Str("version", version).Err(err).Msg("failed to create cpe for Terraform provider")
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

// ParseProviderSource parses a provider source address like
// "registry.terraform.io/hashicorp/aws" into namespace and type.
// Strips the default registry prefix if present. Handles custom registries
// (e.g., "custom.registry.io/namespace/type") by detecting the hostname.
func ParseProviderSource(source string) (namespace, providerType string) {
	// Strip default registry prefix
	source = strings.TrimPrefix(source, defaultRegistry)

	parts := strings.Split(source, "/")
	switch len(parts) {
	case 3:
		// registry/namespace/type (custom registry, not yet stripped)
		return parts[1], parts[2]
	case 2:
		// namespace/type (default registry already stripped, or no registry)
		return parts[0], parts[1]
	default:
		return "", source
	}
}
