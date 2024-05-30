// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pack

import (
	_ "embed"
	"go.mondoo.com/cnquery/v11/explorer"
)

// SBOMQueryPack is a protobuf message that contains the SBOM query pack
//
//go:embed sbom.mql.yaml
var sbomQueryPack []byte

func QueryPack() (*explorer.Bundle, error) {
	return explorer.BundleFromYAML(sbomQueryPack)
}
