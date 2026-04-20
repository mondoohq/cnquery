// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cabalfreeze

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCabalFreezeExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.cabal.project.freeze")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "cabal.project.freeze")
	require.NoError(t, err)

	assert.Nil(t, info.Root())
	assert.Nil(t, info.Direct())

	transitive := info.Transitive()
	// 5 packages (aeson, base, bytestring, text, containers)
	// "aeson +ordered-keymap" (flag, no ==) is skipped
	assert.Equal(t, 5, len(transitive))

	p := transitive.Find("aeson")
	require.NotNil(t, p)
	assert.Equal(t, "2.2.1.0", p.Version)
	assert.Equal(t, "pkg:hackage/aeson@2.2.1.0", p.Purl)

	p = transitive.Find("base")
	require.NotNil(t, p)
	assert.Equal(t, "4.19.0.0", p.Version)

	p = transitive.Find("containers")
	require.NotNil(t, p)
	assert.Equal(t, "0.6.8", p.Version)
}
