// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtobomSpdxJsonDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.spdx.json")
	require.NoError(t, err)

	decoder := NewProtobom()

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)

}

func TestProtobomCycloneDxJsonDecoder(t *testing.T) {
	f, err := os.Open("testdata/alpine-319.cyclone.json")
	require.NoError(t, err)

	decoder := NewProtobom()

	sbomReport, err := decoder.Parse(f)
	require.NoError(t, err)
	assert.NotNil(t, sbomReport)
}
