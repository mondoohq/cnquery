// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSpdx(t *testing.T) {
	r, err := NewReportCollectionJsonFromSingleFile("./testdata/alpine.json")
	require.NoError(t, err)

	sboms, err := GenerateBom(r)
	require.NoError(t, err)

	// store bom in different formats
	selectedBom := sboms[0]

	var exporter Exporter
	output := bytes.Buffer{}
	exporter = &Spdx{
		Version: "2.3",
		Format:  FormatSpdxJSON,
	}
	err = exporter.Render(&output, &selectedBom)
	require.NoError(t, err)

	res := output.String()
	assert.Contains(t, res, "SPDX-2.3")
	assert.Contains(t, res, "\"name\": \"alpine-baselayout\",")
	assert.Contains(t, res, "\"cpe:2.3:a:alpine-baselayout:alpine-baselayout:1695795276:aarch64:*:*:*:*:*:*\"")
}
