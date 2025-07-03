// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"io"
)

type CnqueryBOM struct {
	opts renderOpts
}

func (s *CnqueryBOM) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (ccx *CnqueryBOM) Convert(bom *Sbom) (interface{}, error) {
	// nothing to do, the cnquery BOM is already in the correct format
	return bom, nil
}

func (ccx *CnqueryBOM) Render(output io.Writer, bom *Sbom) error {
	if !ccx.opts.IncludeEvidence {
		// if we do not include evidence, we remove all evidence from the BOM
		for _, pkg := range bom.Packages {
			pkg.EvidenceList = nil
		}
	}

	if !ccx.opts.IncludeCPE {
		// if we do not include CPE, we remove all CPE from the BOM
		for _, pkg := range bom.Packages {
			pkg.Cpes = nil
		}

		if bom.Asset != nil && bom.Asset.Platform != nil {
			bom.Asset.Platform.Cpes = nil
		}
	}

	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	return enc.Encode(bom)
}

func (ccx *CnqueryBOM) Parse(r io.Reader) (*Sbom, error) {
	var s Sbom
	err := json.NewDecoder(r).Decode(&s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
