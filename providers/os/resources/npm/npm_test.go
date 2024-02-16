// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func findPkg(pkgs []*Package, name string) *Package {
	for _, p := range pkgs {
		if p.Name == name {
			return p
		}
	}
	panic("package not found")
}

func TestPurlAndCPE(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		expectedPurl string
		expectedCpes []string
	}{
		{
			name:         "test",
			version:      "1.0.0",
			expectedPurl: "pkg:npm/test@1.0.0",
			expectedCpes: []string{"cpe:2.3:a:test:test:1.0.0:*:*:*:*:*:*:*"},
		},
		{
			name:         "@coreui/vue",
			version:      "2.1.2",
			expectedPurl: "pkg:npm/%40coreui/vue@2.1.2",
			expectedCpes: []string{"cpe:2.3:a:\\@coreui\\/vue:\\@coreui\\/vue:2.1.2:*:*:*:*:*:*:*"},
		},
		{
			name:         "@babel/runtime",
			version:      "7.22.6",
			expectedPurl: "pkg:npm/%40babel/runtime@7.22.6",
			expectedCpes: []string{"cpe:2.3:a:\\@babel\\/runtime:\\@babel\\/runtime:7.22.6:*:*:*:*:*:*:*"},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			purl := NewPackageUrl(test.name, test.version)
			assert.Equal(t, test.expectedPurl, purl)

			cpes := NewCpes(test.name, test.version)
			assert.Equal(t, test.expectedCpes, cpes)
		})
	}
}
