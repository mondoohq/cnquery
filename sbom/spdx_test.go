// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/cnquery/v11/sbom/generator"
)

func TestSpdxOutput(t *testing.T) {
	report, err := generator.LoadReport("./testdata/alpine.json")
	require.NoError(t, err)

	sboms := generator.GenerateBom(report)

	// store bom in different formats
	selectedBom := sboms[0]

	formatHandler := &sbom.Spdx{
		Version: "2.3",
		Format:  sbom.FormatSpdxJSON,
	}

	output := bytes.Buffer{}
	err = formatHandler.Render(&output, selectedBom)
	require.NoError(t, err)

	data := output.String()
	assert.Contains(t, data, "SPDX-2.3")

	// ensure os package is included
	assert.Contains(t, data, "alpine-baselayout")
	assert.Contains(t, data, "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1695795276:aarch64:*:*:*:*:*:*")

	// ensure python package is included
	assert.Contains(t, data, "pip")
	assert.Contains(t, data, "cpe:2.3:a:pip_project:pip:21.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:pypi/pip@21.2.4")

	// ensure npm package is included
	assert.Contains(t, data, "npm")
	assert.Contains(t, data, "cpe:2.3:a:npm:npm:10.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:npm/npm@10.2.4")
}

func TestTagValueDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.spdx")
	require.NoError(t, err)

	decoder := sbom.NewSPDX(sbom.FormatSpdxTagValue)

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)
}

func TestJsonDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.spdx.json")
	require.NoError(t, err)

	decoder := sbom.NewSPDX(sbom.FormatSpdxJSON)

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)
}
