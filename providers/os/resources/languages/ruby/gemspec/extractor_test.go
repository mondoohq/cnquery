// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gemspec

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGemspecExtractor(t *testing.T) {
	f, err := os.Open("./testdata/example.gemspec")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/inspec.gemspec")
	require.NoError(t, err)

	// Root
	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "inspec", root.Name)
	assert.Equal(t, "6.8.1", root.Version)
	assert.Equal(t, "pkg:gem/inspec@6.8.1", root.Purl)

	// Direct = non-dev deps
	direct := info.Direct()
	assert.Equal(t, 3, len(direct))
	assert.NotNil(t, direct.Find("train"))
	assert.NotNil(t, direct.Find("rake"))
	assert.NotNil(t, direct.Find("mongo"))
	assert.Nil(t, direct.Find("minitest")) // dev dep excluded

	// Transitive = all declared deps
	transitive := info.Transitive()
	assert.Equal(t, 4, len(transitive))
	assert.NotNil(t, transitive.Find("minitest"))

	// Exact pin "= 2.13.2" — versioned PURL with extracted version
	mongo := direct.Find("mongo")
	require.NotNil(t, mongo)
	assert.Equal(t, "= 2.13.2", mongo.Version)
	assert.Equal(t, "pkg:gem/mongo@2.13.2", mongo.Purl)

	// Range constraint — versionless PURL
	train := direct.Find("train")
	require.NotNil(t, train)
	assert.Equal(t, "~> 3.10", train.Version)
	assert.Equal(t, "pkg:gem/train", train.Purl) // versionless

	// No constraint — versionless PURL
	rake := direct.Find("rake")
	require.NotNil(t, rake)
	assert.Equal(t, "", rake.Version)
	assert.Equal(t, "pkg:gem/rake", rake.Purl)
}
