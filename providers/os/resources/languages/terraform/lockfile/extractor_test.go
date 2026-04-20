// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package lockfile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestTerraformLockExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.terraform.lock.hcl")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, ".terraform.lock.hcl")
	require.NoError(t, err)

	assert.Nil(t, info.Root())
	assert.Nil(t, info.Direct())

	transitive := info.Transitive()
	assert.Equal(t, 3, len(transitive))

	p := transitive.Find("hashicorp/aws")
	require.NotNil(t, p)
	assert.Equal(t, "5.31.0", p.Version)
	assert.Equal(t, "pkg:terraform/hashicorp/aws@5.31.0", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: ".terraform.lock.hcl"}}, p.EvidenceList)

	p = transitive.Find("hashicorp/random")
	require.NotNil(t, p)
	assert.Equal(t, "3.6.0", p.Version)
	assert.Equal(t, "pkg:terraform/hashicorp/random@3.6.0", p.Purl)

	// Non-hashicorp provider
	p = transitive.Find("integrations/github")
	require.NotNil(t, p)
	assert.Equal(t, "5.42.0", p.Version)
	assert.Equal(t, "pkg:terraform/integrations/github@5.42.0", p.Purl)
}
