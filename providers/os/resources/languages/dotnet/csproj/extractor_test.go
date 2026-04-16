// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package csproj

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsprojExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.csproj")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/MyApp.csproj")
	require.NoError(t, err)

	assert.Nil(t, info.Root())

	// Direct excludes PrivateAssets="all" packages
	direct := info.Direct()
	assert.Equal(t, 2, len(direct))
	assert.NotNil(t, direct.Find("Newtonsoft.Json"))
	assert.NotNil(t, direct.Find("Serilog"))
	assert.Nil(t, direct.Find("xunit"))

	// Transitive includes all
	transitive := info.Transitive()
	assert.Equal(t, 4, len(transitive))
	assert.NotNil(t, transitive.Find("coverlet.collector"))
	assert.NotNil(t, transitive.Find("xunit"))

	p := transitive.Find("Newtonsoft.Json")
	require.NotNil(t, p)
	assert.Equal(t, "13.0.3", p.Version)
	assert.Equal(t, "pkg:nuget/Newtonsoft.Json@13.0.3", p.Purl)
}

func TestCsprojExtractorChildElements(t *testing.T) {
	f, err := os.Open("./testdata/child-elements.csproj")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/MyApp.csproj")
	require.NoError(t, err)

	// Version specified as child element should be parsed
	direct := info.Direct()
	assert.Equal(t, 1, len(direct))
	p := direct.Find("Newtonsoft.Json")
	require.NotNil(t, p)
	assert.Equal(t, "13.0.3", p.Version)
	assert.Equal(t, "pkg:nuget/Newtonsoft.Json@13.0.3", p.Purl)

	// PrivateAssets as child element should mark as dev
	assert.Nil(t, direct.Find("xunit"))

	transitive := info.Transitive()
	assert.Equal(t, 2, len(transitive))
	assert.NotNil(t, transitive.Find("xunit"))
}
