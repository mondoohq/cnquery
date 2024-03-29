// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"io"
)

type CnqueryBOM struct{}

func (ccx *CnqueryBOM) Convert(bom *Sbom) (interface{}, error) {
	// nothing to convert
	return bom, nil
}

func (ccx *CnqueryBOM) Render(output io.Writer, bom *Sbom) error {
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
