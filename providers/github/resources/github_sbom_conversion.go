// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	sbompkg "go.mondoo.com/mql/v13/sbom"
)

// GithubSbomToMqlSbom converts a github.repository.sbom MQL resource to the
// Mondoo SBOM proto format.
func GithubSbomToMqlSbom(s *mqlGithubRepositorySbom) *sbompkg.Sbom {
	if s == nil {
		return nil
	}

	result := &sbompkg.Sbom{
		Status:    sbompkg.Status_STATUS_SUCCEEDED,
		Generator: &sbompkg.Generator{Name: "github-dependency-graph"},
	}

	// Asset: use the SBOM document name (e.g., "owner/repo")
	if s.Name.Error == nil {
		result.Asset = &sbompkg.Asset{Name: s.Name.Data}
	}

	// Timestamp and generator tool name come from creationInfo
	if s.CreationInfo.Error == nil && s.CreationInfo.Data != nil {
		if ci, ok := s.CreationInfo.Data.(map[string]any); ok {
			if created, ok := ci["created"].(string); ok {
				result.Timestamp = created
			}
			if creators, ok := ci["creators"].([]any); ok {
				for _, c := range creators {
					if cs, ok := c.(string); ok && strings.HasPrefix(cs, "Tool: ") {
						result.Generator.Name = strings.TrimPrefix(cs, "Tool: ")
						break
					}
				}
			}
		}
	}

	// Packages
	if s.Packages.Error == nil {
		for _, p := range s.Packages.Data {
			pkg, ok := p.(*mqlGithubRepositorySbomPackage)
			if !ok {
				continue
			}
			result.Packages = append(result.Packages, githubSbomPackage(pkg))
		}
	}

	return result
}

func githubSbomPackage(p *mqlGithubRepositorySbomPackage) *sbompkg.Package {
	pkg := &sbompkg.Package{}

	if p.Name.Error == nil {
		pkg.Name = p.Name.Data
	}
	if p.VersionInfo.Error == nil {
		pkg.Version = p.VersionInfo.Data
	}
	if p.Supplier.Error == nil {
		pkg.Vendor = p.Supplier.Data
	}
	if p.DownloadLocation.Error == nil {
		pkg.Location = p.DownloadLocation.Data
	}

	// Extract purl from external refs; derive package type from it.
	// GitHub SBOM external refs use referenceCategory "PACKAGE-MANAGER" and
	// referenceType "purl" for package URLs.
	if p.ExternalRefs.Error == nil {
		for _, ref := range p.ExternalRefs.Data {
			m, ok := ref.(map[string]any)
			if !ok {
				continue
			}
			refType, _ := m["referenceType"].(string)
			locator, _ := m["referenceLocator"].(string)
			if strings.EqualFold(refType, "purl") && locator != "" {
				pkg.Purl = locator
				// Derive type from purl scheme: "pkg:<type>/..." → e.g., "npm", "pypi"
				if after, found := strings.CutPrefix(locator, "pkg:"); found {
					if idx := strings.IndexAny(after, "/@"); idx > 0 {
						pkg.Type = after[:idx]
					}
				}
				break
			}
		}
	}

	return pkg
}
