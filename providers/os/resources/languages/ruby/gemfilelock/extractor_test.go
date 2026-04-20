// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gemfilelock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestGemfileLockExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.Gemfile.lock")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/Gemfile.lock")
	require.NoError(t, err)

	assert.Nil(t, info.Root())

	// Direct dependencies (from DEPENDENCIES section)
	direct := info.Direct()
	assert.Equal(t, 3, len(direct))

	p := direct.Find("actioncable")
	require.NotNil(t, p)
	assert.Equal(t, "7.1.3", p.Version)
	assert.Equal(t, "pkg:gem/actioncable@7.1.3", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/Gemfile.lock"}}, p.EvidenceList)

	p = direct.Find("puma")
	require.NotNil(t, p)
	assert.Equal(t, "6.4.2", p.Version)

	p = direct.Find("nokogiri")
	require.NotNil(t, p)
	assert.Equal(t, "1.16.2", p.Version) // platform suffix stripped

	// Transitive deps should NOT be in direct
	assert.Nil(t, direct.Find("rack"))
	assert.Nil(t, direct.Find("nio4r"))

	// Transitive = all gems
	transitive := info.Transitive()
	assert.Equal(t, 8, len(transitive))

	p = transitive.Find("rack")
	require.NotNil(t, p)
	assert.Equal(t, "3.0.8", p.Version)
	assert.Equal(t, "pkg:gem/rack@3.0.8", p.Purl)

	p = transitive.Find("nio4r")
	require.NotNil(t, p)
	assert.Equal(t, "2.7.0", p.Version)

	p = transitive.Find("websocket-extensions")
	require.NotNil(t, p)
	assert.Equal(t, "0.1.5", p.Version)
}

func TestParseGemEntry(t *testing.T) {
	assert.Equal(t, gemEntry{"rack", "3.0.8"}, parseGemEntry("rack (3.0.8)"))
	assert.Equal(t, gemEntry{"nokogiri", "1.16.2"}, parseGemEntry("nokogiri (1.16.2-x86_64-linux)"))
	assert.Equal(t, gemEntry{"puma", "6.4.2"}, parseGemEntry("puma (6.4.2)"))
	assert.Equal(t, gemEntry{}, parseGemEntry("invalid line"))
}
