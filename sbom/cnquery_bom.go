// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"io"
)

type CnqueryBOM struct{}

func (ccx *CnqueryBOM) Render(output io.Writer, bom *Sbom) error {
	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	return enc.Encode(bom)
}
