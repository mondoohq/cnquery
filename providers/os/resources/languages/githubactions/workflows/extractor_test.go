// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package workflows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestWorkflowExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.yml")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, ".github/workflows/ci.yml")
	require.NoError(t, err)

	assert.Nil(t, info.Root())
	assert.Nil(t, info.Direct())

	transitive := info.Transitive()
	// actions/checkout@v4 (deduplicated), actions/setup-go@v5,
	// docker/build-push-action@v5.1.0, github/codeql-action/init@v3,
	// github/codeql-action/analyze@v3
	// Local (./local-action) and Docker (docker://alpine) are excluded
	assert.Equal(t, 5, len(transitive))

	p := transitive.Find("actions/checkout")
	require.NotNil(t, p)
	assert.Equal(t, "v4", p.Version)
	assert.Equal(t, "pkg:github/actions/checkout@v4", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: ".github/workflows/ci.yml"}}, p.EvidenceList)

	p = transitive.Find("docker/build-push-action")
	require.NotNil(t, p)
	assert.Equal(t, "v5.1.0", p.Version)

	// Sub-path action
	p = transitive.Find("github/codeql-action/init")
	require.NotNil(t, p)
	assert.Equal(t, "v3", p.Version)
	// PURL uses repo only (no sub-path)
	assert.Equal(t, "pkg:github/github/codeql-action@v3", p.Purl)

	// Local and Docker actions should be excluded
	for _, pkg := range transitive {
		assert.NotContains(t, pkg.Name, "local-action")
		assert.NotContains(t, pkg.Name, "alpine")
	}
}
