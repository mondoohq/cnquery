// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packageslockjson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestPackagesLockJsonExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.packages.lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/packages.lock.json")
	require.NoError(t, err)

	// No root in packages.lock.json
	assert.Nil(t, info.Root())

	// Direct deps
	direct := info.Direct()
	assert.Equal(t, 2, len(direct))
	p := direct.Find("Newtonsoft.Json")
	require.NotNil(t, p)
	assert.Equal(t, "13.0.3", p.Version)
	assert.Equal(t, "pkg:nuget/Newtonsoft.Json@13.0.3", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/packages.lock.json"}}, p.EvidenceList)

	p = direct.Find("Serilog")
	require.NotNil(t, p)
	assert.Equal(t, "3.1.1", p.Version)

	// Transitive should NOT be in direct
	assert.Nil(t, direct.Find("System.Text.Json"))

	// All deps (direct + transitive)
	transitive := info.Transitive()
	assert.Equal(t, 4, len(transitive))

	p = transitive.Find("System.Text.Json")
	require.NotNil(t, p)
	assert.Equal(t, "8.0.0", p.Version)
	assert.Equal(t, "pkg:nuget/System.Text.Json@8.0.0", p.Purl)
}
