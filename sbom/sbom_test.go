// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSbomParsing(t *testing.T) {
	r, err := NewReportCollectionJsonFromSingleFile("./testdata/alpine.json")
	require.NoError(t, err)

	sboms, err := GenerateBom(r)
	require.NoError(t, err)

	// store bom in different formats
	selectedBom := sboms[0]

	assert.Equal(t, "alpine:latest", selectedBom.Asset.Name)
	assert.Equal(t, "aarch64", selectedBom.Asset.Platform.Arch)
	assert.Equal(t, "alpine", selectedBom.Asset.Platform.Name)
	assert.Equal(t, "3.19.0", selectedBom.Asset.Platform.Version)
	assert.Equal(t, []string{"//platformid.api.mondoo.app/runtime/docker/images/1dc785547989b0db1c3cd9949c57574393e69bea98bfe044b0588e24721aa402"}, selectedBom.Asset.PlatformIds)

	var pkg *Package
	for i := range selectedBom.Packages {
		if selectedBom.Packages[i].Name == "alpine-baselayout" {
			pkg = selectedBom.Packages[i]
			break
		}
	}
	require.NotNil(t, pkg)
	assert.Equal(t, "alpine-baselayout", pkg.Name)
	assert.Contains(t, pkg.Evidences, &Evidence{
		Type:  EvidenceType_EVIDENCE_TYPE_FILE,
		Value: "etc/profile.d/color_prompt.sh.disabled",
	})

}

func TestArnGeneration(t *testing.T) {
	platformID := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/12345678910/regions/us-east-1/instances/i-1234567890abcdef0"
	ids := enrichPlatformIds([]string{platformID})
	assert.Equal(t, 2, len(ids))
	assert.Contains(t, ids, platformID)
	assert.Contains(t, ids, "arn:aws:ec2:us-east-1:12345678910:instance/i-1234567890abcdef0")
}
