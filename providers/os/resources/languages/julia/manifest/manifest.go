// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"io"

	"github.com/BurntSushi/toml"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	julialang "go.mondoo.com/mql/v13/providers/os/resources/languages/julia"
	"go.mondoo.com/mql/v13/sbom"
)

// Extractor parses Julia Manifest.toml files.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "Julia Manifest.toml Extractor"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Try v2 format first (Julia >= 1.7), then v1
	pkgs, err := parseManifestV2(data, filename)
	if err != nil || len(pkgs) == 0 {
		pkgs, err = parseManifestV1(data, filename)
		if err != nil {
			return nil, err
		}
	}

	return &manifestBom{transitive: pkgs}, nil
}

// manifestV2 represents the Manifest.toml v2 format (Julia >= 1.7).
// Structure:
//
//	julia_version = "1.9.0"
//	manifest_format = "2.0"
//	[deps]
//	[deps.HTTP]
//	  [[deps.HTTP]]
//	  uuid = "..."
//	  version = "1.10.1"
type manifestV2 struct {
	ManifestFormat string                         `toml:"manifest_format"`
	Deps           map[string][]manifestV2Package `toml:"deps"`
}

type manifestV2Package struct {
	UUID    string `toml:"uuid"`
	Version string `toml:"version"`
}

func parseManifestV2(data []byte, filename string) (languages.Packages, error) {
	var m manifestV2
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	if m.ManifestFormat == "" || m.Deps == nil {
		return nil, nil
	}

	var pkgs languages.Packages
	for name, entries := range m.Deps {
		for _, entry := range entries {
			if entry.Version == "" {
				continue
			}
			pkgs = append(pkgs, &languages.Package{
				Name:    name,
				Version: entry.Version,
				Purl:    julialang.NewPackageUrl(name, entry.Version),
				EvidenceList: []*sbom.Evidence{
					julialang.NewEvidence(filename),
				},
			})
		}
	}

	return pkgs, nil
}

// manifestV1 represents the Manifest.toml v1 format (Julia < 1.7).
// Structure:
//
//	[[HTTP]]
//	uuid = "..."
//	version = "1.10.1"
type manifestV1Package struct {
	UUID    string `toml:"uuid"`
	Version string `toml:"version"`
}

func parseManifestV1(data []byte, filename string) (languages.Packages, error) {
	var raw map[string][]manifestV1Package
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var pkgs languages.Packages
	for name, entries := range raw {
		// Skip metadata keys
		if name == "julia_version" || name == "manifest_format" {
			continue
		}
		for _, entry := range entries {
			if entry.Version == "" {
				continue
			}
			pkgs = append(pkgs, &languages.Package{
				Name:    name,
				Version: entry.Version,
				Purl:    julialang.NewPackageUrl(name, entry.Version),
				EvidenceList: []*sbom.Evidence{
					julialang.NewEvidence(filename),
				},
			})
		}
	}

	return pkgs, nil
}

type manifestBom struct {
	transitive languages.Packages
}

func (b *manifestBom) Root() *languages.Package {
	return nil
}

func (b *manifestBom) Direct() languages.Packages {
	return nil
}

func (b *manifestBom) Transitive() languages.Packages {
	return b.transitive
}
