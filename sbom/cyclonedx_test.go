// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"bytes"
	"testing"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCycloneDXOutput(t *testing.T) {
	r := loadTestReport(t)
	sboms, err := GenerateBom(r)
	require.NoError(t, err)

	// store bom in different formats
	selectedBom := sboms[0]

	formatHandler := &CycloneDX{
		Format: cyclonedx.BOMFileFormatJSON,
	}

	output := bytes.Buffer{}
	err = formatHandler.Render(&output, selectedBom)
	require.NoError(t, err)
	data := output.String()

	// os.WriteFile("./testdata/bom_cyclone.json", output.Bytes(), 0700)
	assert.Contains(t, data, "cyclonedx")

	// ensure os package is included
	assert.Contains(t, data, "alpine-baselayout")
	assert.Contains(t, data, "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1695795276:aarch64:*:*:*:*:*:*")
	// check that package files are included
	assert.Contains(t, data, "etc/profile.d/color_prompt.sh.disabled")

	// ensure python package is included
	assert.Contains(t, data, "pip")
	assert.Contains(t, data, "cpe:2.3:a:pip_project:pip:21.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:pypi/pip@21.2.4")

	// ensure npm package is included
	assert.Contains(t, data, "npm")
	assert.Contains(t, data, "cpe:2.3:a:npm:npm:10.2.4:*:*:*:*:*:*:*")
	assert.Contains(t, data, "pkg:npm/npm@10.2.4")
}
