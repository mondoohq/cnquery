// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packagejson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/sbom"
)

func TestPackageJsonExtractor(t *testing.T) {
	f, err := os.Open("./testdata/express-package.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/package.json")
	assert.Nil(t, err)

	root := info.Root()

	assert.Equal(t, &sbom.Package{
		Name:         "express",
		Version:      "4.16.4",
		Purl:         "pkg:npm/express@4.16.4",
		Cpes:         []string{"cpe:2.3:a:express:express:4.16.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, root, "express package is not as expected")

	list := info.Transitive()
	assert.Equal(t, 31, len(list))

	// ensure the package is in the list
	p := list.Find("express")
	assert.Equal(t, &sbom.Package{
		Name:         "express",
		Version:      "4.16.4",
		Purl:         "pkg:npm/express@4.16.4",
		Cpes:         []string{"cpe:2.3:a:express:express:4.16.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "express package is not as expected")

	p = list.Find("path-to-regexp")
	assert.Equal(t, &sbom.Package{
		Name:         "path-to-regexp",
		Version:      "0.1.7",
		Purl:         "pkg:npm/path-to-regexp@0.1.7",
		Cpes:         []string{"cpe:2.3:a:path-to-regexp:path-to-regexp:0.1.7:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "path-to-regexp package is not as expected")

	p = list.Find("range-parser")
	assert.Equal(t, &sbom.Package{
		Name:         "range-parser",
		Version:      "~1.2.0",
		Purl:         "pkg:npm/range-parser@1.2.0",
		Cpes:         []string{"cpe:2.3:a:range-parser:range-parser:1.2.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "range-parser package is not as expected")
}
