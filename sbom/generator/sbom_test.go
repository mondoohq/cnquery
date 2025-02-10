// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"testing"

	"go.mondoo.com/cnquery/v11/sbom"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSbomGeneration(t *testing.T) {
	t.Run("generate sbom from a full report", func(t *testing.T) {
		report, err := LoadReport("../testdata/alpine.json")
		require.NoError(t, err)

		sboms, err := GenerateBom(report)
		require.NoError(t, err)

		// store bom in different formats
		selectedBom := sboms[0]

		assert.Equal(t, "alpine:latest", selectedBom.Asset.Name)
		assert.Equal(t, "trace-alpine", selectedBom.Asset.TraceId)
		assert.Equal(t, "aarch64", selectedBom.Asset.Platform.Arch)
		assert.Equal(t, "alpine", selectedBom.Asset.Platform.Name)
		assert.Equal(t, "3.19.0", selectedBom.Asset.Platform.Version)
		assert.Equal(t, []string{"//platformid.api.mondoo.app/runtime/docker/images/1dc785547989b0db1c3cd9949c57574393e69bea98bfe044b0588e24721aa402"}, selectedBom.Asset.PlatformIds)

		// search os package
		pkg := findProtoPkg(selectedBom.Packages, "alpine-baselayout")
		assert.Equal(t, "alpine-baselayout", pkg.Name)
		assert.Contains(t, pkg.EvidenceList, &sbom.Evidence{
			Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
			Value: "etc/profile.d/color_prompt.sh.disabled",
		})

		// search python package
		pkg = findProtoPkg(selectedBom.Packages, "pip")
		assert.Equal(t, "pip", pkg.Name)
		assert.Contains(t, pkg.EvidenceList, &sbom.Evidence{
			Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
			Value: "/opt/lib/python3.9/site-packages/pip-21.2.4.dist-info/METADATA",
		})

		// search npm package
		pkg = findProtoPkg(selectedBom.Packages, "npm")
		assert.Equal(t, "npm", pkg.Name)
		assert.Contains(t, pkg.EvidenceList, &sbom.Evidence{
			Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
			Value: "/opt/lib/node_modules/npm/package.json",
		})
	})
	t.Run("generate sbom from a report with a package error", func(t *testing.T) {
		report, err := LoadReport("testdata/alpine-failed-package.json")
		require.NoError(t, err)

		sboms, err := GenerateBom(report)
		require.NoError(t, err)

		// store bom in different formats
		selectedBom := sboms[0]
		assert.Equal(t, sbom.Status_STATUS_FAILED, selectedBom.Status)
		assert.Contains(t, selectedBom.ErrorMessage, "failed to parse bom fields json data")
	})
}

func findProtoPkg(pkgs []*sbom.Package, name string) *sbom.Package {
	for i := range pkgs {
		if pkgs[i].Name == name {
			return pkgs[i]
		}
	}
	panic("package not found: " + name)
}

func TestArnGeneration(t *testing.T) {
	platformID := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/12345678910/regions/us-east-1/instances/i-1234567890abcdef0"
	ids := enrichPlatformIds([]string{platformID})
	assert.Equal(t, 2, len(ids))
	assert.Contains(t, ids, platformID)
	assert.Contains(t, ids, "arn:aws:ec2:us-east-1:12345678910:instance/i-1234567890abcdef0")
}
