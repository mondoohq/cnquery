// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packagelockjson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/sbom"
)

func TestPackageJsonLockExtractorWithPackages(t *testing.T) {
	f, err := os.Open("./testdata/lockfile-v2.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &sbom.Package{
		Name:         "npm",
		Version:      "7.0.0",
		Purl:         "pkg:npm/npm@7.0.0",
		Cpes:         []string{"cpe:2.3:a:npm:npm:7.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, root)

	list := info.Transitive()
	assert.Equal(t, 2, len(list))

	p := list.Find("@babel/code-frame")
	assert.Equal(t, &sbom.Package{
		Name:         "@babel/code-frame",
		Version:      "7.10.4",
		Purl:         "pkg:npm/node-modules/%40babel@7.10.4",
		Cpes:         []string{"cpe:2.3:a:node_modules\\/\\@babel\\/code-frame:node_modules\\/\\@babel\\/code-frame:7.10.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)

}

func TestPackageJsonLockExtractorWithDependencies(t *testing.T) {
	f, err := os.Open("./testdata/workbox-package-lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &sbom.Package{
		Name:         "workbox",
		Version:      "0.0.0",
		Purl:         "pkg:npm/workbox@0.0.0",
		Cpes:         []string{"cpe:2.3:a:workbox:workbox:0.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, root)

	list := info.Transitive()
	assert.Equal(t, 1299, len(list))

	p := list.Find("@babel/generator")
	assert.Equal(t, &sbom.Package{
		Name:         "@babel/generator",
		Version:      "7.0.0",
		Purl:         "pkg:npm/%40babel/generator@7.0.0",
		Cpes:         []string{"cpe:2.3:a:\\@babel\\/generator:\\@babel\\/generator:7.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)

	p = list.Find("@lerna/changed")
	assert.Equal(t, &sbom.Package{
		Name:         "@lerna/changed",
		Version:      "3.3.2",
		Purl:         "pkg:npm/%40lerna/changed@3.3.2",
		Cpes:         []string{"cpe:2.3:a:\\@lerna\\/changed:\\@lerna\\/changed:3.3.2:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)
}
