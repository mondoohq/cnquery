// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package sbom_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

// graphBom builds a tiny two-package SBOM with a dependency edge and a dev-scoped
// package to exercise bom_ref, the dependency graph, and scope rendering.
func graphBom() *sbom.Sbom {
	return &sbom.Sbom{
		Generator: &sbom.Generator{Name: "test", Version: "1", Vendor: "Mondoo"},
		Asset:     &sbom.Asset{Name: "app", Platform: &sbom.Platform{}},
		Packages: []*sbom.Package{
			{Name: "app", Version: "1.0.0", Type: "npm", Purl: "pkg:npm/app@1.0.0"},
			{Name: "left-pad", Version: "1.3.0", Type: "npm", Purl: "pkg:npm/left-pad@1.3.0"},
			{Name: "jest", Version: "29.0.0", Type: "npm", Purl: "pkg:npm/jest@29.0.0", Scope: sbom.PackageScopeDev},
		},
		Dependencies: []*sbom.Dependency{
			{Ref: "pkg:npm/app@1.0.0", DependsOn: []string{"pkg:npm/left-pad@1.3.0"}},
		},
	}
}

func TestBomRefFor(t *testing.T) {
	// purl-when-present
	assert.Equal(t, "pkg:npm/left-pad@1.3.0",
		sbom.BomRefFor(&sbom.Package{Name: "left-pad", Version: "1.3.0", Type: "npm", Purl: "pkg:npm/left-pad@1.3.0"}))
	// synthesized fallback when no purl
	assert.Equal(t, "npm/left-pad@1.3.0",
		sbom.BomRefFor(&sbom.Package{Name: "left-pad", Version: "1.3.0", Type: "npm"}))
	// an already-set ref wins
	assert.Equal(t, "explicit",
		sbom.BomRefFor(&sbom.Package{Name: "left-pad", Purl: "pkg:npm/left-pad@1.3.0", BomRef: "explicit"}))
}

func TestCycloneDxDependencyGraph(t *testing.T) {
	out := bytes.Buffer{}
	require.NoError(t, sbom.New(sbom.FormatCycloneDxJSON).Render(&out, graphBom()))
	data := out.String()

	// stable bom-ref (the component's purl), not a random UUID
	assert.Contains(t, data, `"bom-ref": "pkg:npm/app@1.0.0"`)
	assert.Contains(t, data, `"bom-ref": "pkg:npm/left-pad@1.3.0"`)
	// dependency graph section, referencing components by bom-ref
	assert.Contains(t, data, `"dependencies"`)
	assert.Contains(t, data, `"ref": "pkg:npm/app@1.0.0"`)
	assert.Contains(t, data, `"pkg:npm/left-pad@1.3.0"`)
	// dev-scoped package rendered as excluded scope
	assert.Contains(t, data, `"scope": "excluded"`)
}

func TestSpdxDependencyGraph(t *testing.T) {
	out := bytes.Buffer{}
	require.NoError(t, sbom.NewSPDX(sbom.FormatSpdxJSON).Render(&out, graphBom()))
	data := out.String()

	// SPDX emits a DEPENDS_ON relationship for the edge
	assert.Contains(t, data, "DEPENDS_ON")
}
