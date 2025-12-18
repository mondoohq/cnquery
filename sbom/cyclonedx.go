// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
	"time"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"github.com/package-url/packageurl-go"
)

func NewCycloneDX(format string) *CycloneDX {
	switch format {
	case FormatCycloneDxXML:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatXML,
		}
	default:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}
	}
}

type CycloneDX struct {
	opts   renderOpts
	Format cyclonedx.BOMFileFormat
}

func (ccx *CycloneDX) convertToCycloneDx(bom *Sbom) (*cyclonedx.BOM, error) {
	sbom := cyclonedx.NewBOM()
	sbom.SerialNumber = uuid.New().URN()
	sbom.Metadata = &cyclonedx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &cyclonedx.ToolsChoice{
			Components: &[]cyclonedx.Component{
				{
					Type:    cyclonedx.ComponentTypeApplication,
					Author:  bom.Generator.Vendor,
					Name:    bom.Generator.Name,
					Version: bom.Generator.Version,
				},
			},
		},
		Component: &cyclonedx.Component{
			BOMRef: uuid.New().String(),
			// TODO: understand the device type
			// Type: cyclonedx.ComponentTypeContainer,
			Type: cyclonedx.ComponentTypeDevice,
			Name: bom.Asset.Name,
		},
	}

	components := []cyclonedx.Component{}

	// add os as component
	cpe := ""
	if len(bom.Asset.Platform.Cpes) > 0 {
		cpe = bom.Asset.Platform.Cpes[0]
	}

	components = append(components, cyclonedx.Component{
		BOMRef:  uuid.New().String(),
		Type:    cyclonedx.ComponentTypeOS,
		Name:    bom.Asset.Platform.Name,
		Version: bom.Asset.Platform.Version,
		CPE:     cpe,
	})

	// add os packages as components
	for i := range bom.Packages {
		pkg := bom.Packages[i]
		cpe := ""
		if len(pkg.Cpes) > 0 && ccx.opts.IncludeCPE {
			cpe = pkg.Cpes[0]
		}

		fileLocations := []cyclonedx.EvidenceOccurrence{}

		// pkg.Location is deprecated, use pkg.Evidences instead
		if pkg.Location != "" {
			fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
				Location: pkg.Location,
			})
		}

		if pkg.EvidenceList != nil && ccx.opts.IncludeEvidence {
			for i := range pkg.EvidenceList {
				e := pkg.EvidenceList[i]
				if e.Type == EvidenceType_EVIDENCE_TYPE_FILE {
					fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
						Location: e.Value,
					})
				}
			}
		}

		var evidence *cyclonedx.Evidence
		if len(fileLocations) > 0 {
			evidence = &cyclonedx.Evidence{
				Occurrences: &fileLocations,
			}
		}

		bomPkg := cyclonedx.Component{
			BOMRef:     uuid.New().String(), // temporary, we need to store the relationships next
			Type:       cyclonedx.ComponentTypeLibrary,
			Name:       pkg.Name,
			Version:    pkg.Version,
			PackageURL: pkg.Purl,
			CPE:        cpe,
			Evidence:   evidence,
		}

		components = append(components, bomPkg)
	}

	sbom.Components = &components

	return sbom, nil
}

func (s *CycloneDX) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (ccx *CycloneDX) Convert(bom *Sbom) (any, error) {
	return ccx.convertToCycloneDx(bom)
}

func (ccx *CycloneDX) Render(w io.Writer, bom *Sbom) error {
	sbom, err := ccx.convertToCycloneDx(bom)
	if err != nil {
		return err
	}
	enc := cyclonedx.NewBOMEncoder(w, ccx.Format)
	enc.SetPretty(true)
	return enc.Encode(sbom)
}

func (ccx *CycloneDX) Parse(r io.Reader) (*Sbom, error) {
	doc := &cyclonedx.BOM{
		Components: &[]cyclonedx.Component{},
	}
	err := cyclonedx.NewBOMDecoder(r, ccx.Format).Decode(doc)
	if err != nil {
		return nil, err
	}

	return ccx.convertCycloneDxToSbom(doc)
}

func (ccx *CycloneDX) convertCycloneDxToSbom(bom *cyclonedx.BOM) (*Sbom, error) {
	if bom == nil {
		return nil, nil
	}

	// check if the BOM is empty
	if bom.Metadata == nil || bom.Metadata.Component == nil || bom.Components == nil {
		return nil, errors.New("not a valid cyclone dx BOM")
	}

	sbom := &Sbom{
		Asset: &Asset{
			Name: bom.Metadata.Component.Name + ":" + bom.Metadata.Component.Version,
		},
		Packages: make([]*Package, 0),
	}

	if bom.Metadata.Tools != nil && bom.Metadata.Tools.Components != nil {
		// last one wins :-) - we only support one tool
		for i := range *bom.Metadata.Tools.Components {
			component := (*bom.Metadata.Tools.Components)[i]
			sbom.Generator = &Generator{
				Name:    component.Name,
				Version: component.Version,
				Vendor:  component.Author,
			}
		}
	}

	if bom.Components == nil {
		return sbom, nil
	}

	for i := range *bom.Components {
		component := (*bom.Components)[i]
		pkg := &Package{
			Name:    component.Name,
			Version: component.Version,
			Purl:    component.PackageURL,
		}

		// parse purl to gather package type
		if component.PackageURL != "" {
			url, err := packageurl.FromString(component.PackageURL)
			if err == nil {
				pkg.Type = url.Type
			}
		}

		if component.CPE != "" {
			pkg.Cpes = []string{component.CPE}
		}

		if component.Evidence != nil && component.Evidence.Occurrences != nil && ccx.opts.IncludeEvidence {
			pkg.EvidenceList = make([]*Evidence, 0)
			for i := range *component.Evidence.Occurrences {
				e := (*component.Evidence.Occurrences)[i]
				pkg.EvidenceList = append(pkg.EvidenceList, &Evidence{
					Type:  EvidenceType_EVIDENCE_TYPE_FILE,
					Value: e.Location,
				})
			}
		}

		switch component.Type {
		case cyclonedx.ComponentTypeOS:
			sbom.Asset.Platform = &Platform{
				Name:    component.Name,
				Version: component.Version,
				Title:   component.Description,
			}
			sbom.Asset.Platform.Family = familyMap[component.Name]

			if len(component.CPE) > 0 {
				sbom.Asset.Platform.Cpes = []string{component.CPE}
			}
		case cyclonedx.ComponentTypeLibrary:
			sbom.Packages = append(sbom.Packages, pkg)
		}
	}

	return sbom, nil
}

var familyMap = map[string][]string{
	"windows": {"windows", "os"},
	"macos":   {"darwin", "bsd", "unix", "os"},
	"debian":  {"linux", "unix", "os"},
	"ubuntu":  {"linux", "unix", "os"},
	"centos":  {"linux", "unix", "os"},
	"alpine":  {"linux", "unix", "os"},
	"fedora":  {"linux", "unix", "os"},
	"rhel":    {"linux", "unix", "os"},
}
