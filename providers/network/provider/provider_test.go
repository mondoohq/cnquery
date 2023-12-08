// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTarget(t *testing.T) {
	testCases := []struct {
		name           string
		target         string
		expectedHost   string
		expectedPort   int
		expectedScheme string
		expectedPath   string
		expectError    bool
	}{
		{
			name:           "Normal HTTP URL",
			target:         "http://example.com/path",
			expectedHost:   "example.com",
			expectedPort:   80,
			expectedScheme: "http",
			expectedPath:   "/path",
			expectError:    false,
		},
		{
			name:           "Normal HTTPS URL",
			target:         "https://example.com",
			expectedHost:   "example.com",
			expectedPort:   443,
			expectedScheme: "https",
			expectedPath:   "",
			expectError:    false,
		},
		{
			name:           "URL without scheme",
			target:         "example.com",
			expectedHost:   "example.com",
			expectedPort:   0,
			expectedScheme: "",
			expectedPath:   "",
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host, port, scheme, path, err := parseTarget(tc.target)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedHost, host)
				assert.Equal(t, tc.expectedPort, port)
				assert.Equal(t, tc.expectedScheme, scheme)
				assert.Equal(t, tc.expectedPath, path)
			}
		})
	}
}
