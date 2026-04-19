// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package renvlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRenvLock(t *testing.T) {
	f, err := os.Open("testdata/renv.lock")
	require.NoError(t, err)
	defer f.Close()

	extractor := &Extractor{}
	bom, err := extractor.Parse(f, "testdata/renv.lock")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	require.Len(t, pkgs, 4)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "1.1.4", byName["dplyr"])
	assert.Equal(t, "3.5.0", byName["ggplot2"])
	assert.Equal(t, "1.3.1", byName["tidyr"])
	assert.Equal(t, "1.5.1", byName["stringr"])

	// Check PURL
	dplyr := pkgs.Find("dplyr")
	require.NotNil(t, dplyr)
	assert.Equal(t, "pkg:cran/dplyr@1.1.4", dplyr.Purl)

	// Check evidence
	assert.Len(t, dplyr.EvidenceList, 1)
	assert.Equal(t, "testdata/renv.lock", dplyr.EvidenceList[0].Value)
}
