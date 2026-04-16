// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package depsjson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepsJsonExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.deps.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/MyApp.deps.json")
	require.NoError(t, err)

	assert.Nil(t, info.Root())
	assert.Nil(t, info.Direct())

	// Only packages, not project references
	transitive := info.Transitive()
	assert.Equal(t, 3, len(transitive))

	p := transitive.Find("Newtonsoft.Json")
	require.NotNil(t, p)
	assert.Equal(t, "13.0.3", p.Version)
	assert.Equal(t, "pkg:nuget/Newtonsoft.Json@13.0.3", p.Purl)

	p = transitive.Find("Serilog")
	require.NotNil(t, p)
	assert.Equal(t, "3.1.1", p.Version)

	// Project reference should be excluded
	assert.Nil(t, transitive.Find("MyApp"))
}
