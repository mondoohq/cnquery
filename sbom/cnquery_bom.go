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
	if ccx.opts.RenderWithEvidence == false {
		// if we do not render with evidence, we remove all evidence from the BOM
		for _, pkg := range bom.Packages {
			pkg.EvidenceList = nil
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

	if ccx.opts.RenderWithEvidence == false {
		// if we do not render with evidence, we remove all evidence from the BOM
		for _, pkg := range s.Packages {
			pkg.EvidenceList = nil
		}
	}

	return &s, nil
}
