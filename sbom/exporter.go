// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"io"
)

type Exporter interface {
	Render(w io.Writer, bom *Sbom) error
}
