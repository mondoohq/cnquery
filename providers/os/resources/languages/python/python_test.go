// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			expectedPurl: "pkg:pypi/test@1.0.0",
			expectedCpes: []string{"cpe:2.3:a:test_project:test:1.0.0:*:*:*:*:*:*:*"},
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
