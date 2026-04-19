// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifestV2(t *testing.T) {
	f, err := os.Open("testdata/Manifest_v2.toml")
	require.NoError(t, err)
	defer f.Close()

	extractor := &Extractor{}
	bom, err := extractor.Parse(f, "testdata/Manifest_v2.toml")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	require.Len(t, pkgs, 4)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "1.10.1", byName["HTTP"])
	assert.Equal(t, "0.21.4", byName["JSON"])
	assert.Equal(t, "1.6.1", byName["DataFrames"])
	assert.Equal(t, "1.40.1", byName["Plots"])

	http := pkgs.Find("HTTP")
	require.NotNil(t, http)
	assert.Equal(t, "pkg:julia/HTTP@1.10.1", http.Purl)
	assert.Len(t, http.EvidenceList, 1)
}

func TestParseManifestV1(t *testing.T) {
	f, err := os.Open("testdata/Manifest_v1.toml")
	require.NoError(t, err)
	defer f.Close()

	extractor := &Extractor{}
	bom, err := extractor.Parse(f, "testdata/Manifest_v1.toml")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	// Dates has no version, so should be skipped
	require.Len(t, pkgs, 2)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "1.10.1", byName["HTTP"])
	assert.Equal(t, "0.21.4", byName["JSON"])
}
