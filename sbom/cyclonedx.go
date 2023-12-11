// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"io"
	"time"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
)

type CycloneDX struct {
	Format cyclonedx.BOMFileFormat
}

func (ccx *CycloneDX) convert(bom *Sbom) (*cyclonedx.BOM, error) {
	sbom := cyclonedx.NewBOM()
	sbom.SerialNumber = uuid.New().URN()
	sbom.Metadata = &cyclonedx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &cyclonedx.ToolsChoice{
			Components: &[]cyclonedx.Component{
				{
					Author:  bom.Generator.Vendor,
					Name:    bom.Generator.Name,
					Version: bom.Generator.Version,
				},
			},
		},
		Component: &cyclonedx.Component{
			// BOMRef:  string(bomRef),
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
		Type:    cyclonedx.ComponentTypeOS,
		Name:    bom.Asset.Platform.Name,
		Version: bom.Asset.Platform.Version,
		CPE:     cpe,
	})

	// add os packages as components
	for i := range bom.Packages {
		pkg := bom.Packages[i]
		cpe := ""
		if len(pkg.Cpes) > 0 {
			cpe = pkg.Cpes[0]
		}
		bomPkg := cyclonedx.Component{
			Type:       cyclonedx.ComponentTypeLibrary,
			Name:       pkg.Name,
			Version:    pkg.Version,
			PackageURL: pkg.Purl,
			CPE:        cpe,
			// TODO: use evidence location in 1.5 once available
			Description: pkg.Location,
		}

		components = append(components, bomPkg)
	}

	sbom.Components = &components

	return sbom, nil
}

func (ccx *CycloneDX) Render(w io.Writer, bom *Sbom) error {
	sbom, err := ccx.convert(bom)
	if err != nil {
		return err
	}
	enc := cyclonedx.NewBOMEncoder(w, ccx.Format)
	enc.SetPretty(true)
	return enc.Encode(sbom)
}
