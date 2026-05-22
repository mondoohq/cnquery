// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagejson

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/sbom"
)

func TestPackageJsonExtractor(t *testing.T) {
	f, err := os.Open("./testdata/express-package.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/package.json")
	assert.Nil(t, err)

	root := info.Root()

	assert.Equal(t, &languages.Package{
		Name:         "express",
		Version:      "4.16.4",
		Description:  "Fast, unopinionated, minimalist web framework",
		Author:       "TJ Holowaychuk",
		License:      "MIT",
		Purl:         "pkg:npm/express@4.16.4",
		Cpes:         []string{"cpe:2.3:a:express:express:4.16.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, root, "express package is not as expected")

	list := info.Transitive()
	assert.Equal(t, 31, len(list))

	// ensure the package is in the list — Find returns the root entry,
	// which carries the package.json description/author/license.
	p := list.Find("express")
	assert.Equal(t, &languages.Package{
		Name:         "express",
		Version:      "4.16.4",
		Description:  "Fast, unopinionated, minimalist web framework",
		Author:       "TJ Holowaychuk",
		License:      "MIT",
		Purl:         "pkg:npm/express@4.16.4",
		Cpes:         []string{"cpe:2.3:a:express:express:4.16.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "express package is not as expected")

	p = list.Find("path-to-regexp")
	assert.Equal(t, &languages.Package{
		Name:         "path-to-regexp",
		Version:      "0.1.7",
		Purl:         "pkg:npm/path-to-regexp@0.1.7",
		Cpes:         []string{"cpe:2.3:a:path-to-regexp:path-to-regexp:0.1.7:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "path-to-regexp package is not as expected")

	p = list.Find("range-parser")
	assert.Equal(t, &languages.Package{
		Name:         "range-parser",
		Version:      "~1.2.0",
		Purl:         "pkg:npm/range-parser@1.2.0",
		Cpes:         []string{"cpe:2.3:a:range-parser:range-parser:1.2.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/package.json"}},
	}, p, "range-parser package is not as expected")
}

func TestPackageJsonExtractorAuthorForms(t *testing.T) {
	// package.json supports two `author` shapes — string ("Name <email>")
	// and object ({"name":..., "email":..., "url":...}). Both must
	// surface as the bare name on languages.Package.Author.
	cases := []struct {
		name     string
		raw      string
		wantName string
	}{
		{
			name:     "string with email and url",
			raw:      `{"name":"x","version":"1","author":"Jane Doe <jane@example.com> (https://example.com)"}`,
			wantName: "Jane Doe",
		},
		{
			name:     "string with name only",
			raw:      `{"name":"x","version":"1","author":"Jane Doe"}`,
			wantName: "Jane Doe",
		},
		{
			name:     "object form",
			raw:      `{"name":"x","version":"1","author":{"name":"Jane Doe","email":"jane@example.com"}}`,
			wantName: "Jane Doe",
		},
		{
			name:     "missing author",
			raw:      `{"name":"x","version":"1"}`,
			wantName: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := (&Extractor{}).Parse(strings.NewReader(tc.raw), "p/package.json")
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, info.Root().Author)
		})
	}
}
