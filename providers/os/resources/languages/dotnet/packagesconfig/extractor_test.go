// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagesconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagesConfigExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.packages.config")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/packages.config")
	require.NoError(t, err)

	assert.Nil(t, info.Root())

	// Direct excludes dev dependencies
	direct := info.Direct()
	assert.Equal(t, 2, len(direct))
	assert.NotNil(t, direct.Find("Newtonsoft.Json"))
	assert.NotNil(t, direct.Find("Castle.Core"))
	assert.Nil(t, direct.Find("NUnit"))

	// Transitive includes all
	transitive := info.Transitive()
	assert.Equal(t, 4, len(transitive))
	assert.NotNil(t, transitive.Find("NUnit"))
	assert.NotNil(t, transitive.Find("Moq"))

	p := transitive.Find("Newtonsoft.Json")
	require.NotNil(t, p)
	assert.Equal(t, "13.0.3", p.Version)
	assert.Equal(t, "pkg:nuget/Newtonsoft.Json@13.0.3", p.Purl)
}
