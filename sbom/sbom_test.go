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

	assert.Equal(t, "alpine:3.18", selectedBom.Asset.Name)
	assert.Equal(t, "amd64", selectedBom.Asset.Platform.Arch)
	assert.Equal(t, "alpine", selectedBom.Asset.Platform.Name)
	assert.Equal(t, "3.18.4", selectedBom.Asset.Platform.Version)
	assert.Equal(t, []string{"//platformid.api.mondoo.app/runtime/docker/images/e6b39dab7a69cfea9941378c0dbcc21b314c34eb22f5b9032c2023f6398e97b1"}, selectedBom.Asset.PlatformIds)
}

func TestArnGeneration(t *testing.T) {
	platformID := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/12345678910/regions/us-east-1/instances/i-1234567890abcdef0"
	ids := enrichPlatformIds([]string{platformID})
	assert.Equal(t, 2, len(ids))
	assert.Contains(t, ids, platformID)
	assert.Contains(t, ids, "arn:aws:ec2:us-east-1:12345678910:instance/i-1234567890abcdef0")
}
