// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package composerjson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposerJsonExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.composer.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/composer.json")
	require.NoError(t, err)

	// Root project
	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "myvendor/myproject", root.Name)
	assert.Equal(t, "1.0.0", root.Version)
	assert.Equal(t, "pkg:composer/myvendor/myproject@1.0.0", root.Purl)

	// Direct = require (excluding php, ext-*)
	direct := info.Direct()
	assert.Equal(t, 2, len(direct))
	assert.NotNil(t, direct.Find("monolog/monolog"))
	assert.NotNil(t, direct.Find("symfony/console"))
	// php and ext-json should be excluded
	assert.Nil(t, direct.Find("php"))
	assert.Nil(t, direct.Find("ext-json"))

	// Transitive = require + require-dev
	transitive := info.Transitive()
	assert.Equal(t, 3, len(transitive))
	assert.NotNil(t, transitive.Find("phpunit/phpunit"))
}
