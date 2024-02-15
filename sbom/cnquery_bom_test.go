// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleBom(t *testing.T) {
	r, err := NewReportCollectionJsonFromSingleFile("./testdata/alpine.json")
	require.NoError(t, err)

	sboms, err := GenerateBom(r)
	require.NoError(t, err)

	// store bom in different formats
	selectedBom := sboms[0]

	var exporter Exporter

	output := bytes.Buffer{}
	exporter = &CnqueryBOM{}
	err = exporter.Render(&output, &selectedBom)
	require.NoError(t, err)

	data := output.String()
	assert.Contains(t, data, "alpine-baselayout")
	assert.Contains(t, data, "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1695795276:aarch64:*:*:*:*:*:*")
	// check that package files are included
	assert.Contains(t, data, "etc/profile.d/color_prompt.sh.disabled")
}
