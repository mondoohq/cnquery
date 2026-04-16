// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package composerlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestComposerLockExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.composer.lock")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/composer.lock")
	require.NoError(t, err)

	assert.Nil(t, info.Root())

	// Direct = production packages only
	direct := info.Direct()
	assert.Equal(t, 3, len(direct))

	p := direct.Find("monolog/monolog")
	require.NotNil(t, p)
	assert.Equal(t, "3.5.0", p.Version)
	assert.Equal(t, "pkg:composer/monolog/monolog@3.5.0", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/composer.lock"}}, p.EvidenceList)

	p = direct.Find("symfony/console")
	require.NotNil(t, p)
	assert.Equal(t, "v6.4.1", p.Version)

	// Dev packages should NOT be in direct
	assert.Nil(t, direct.Find("phpunit/phpunit"))

	// Transitive = all packages
	transitive := info.Transitive()
	assert.Equal(t, 5, len(transitive))
	assert.NotNil(t, transitive.Find("phpunit/phpunit"))
	assert.NotNil(t, transitive.Find("mockery/mockery"))
}
