// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAndValidateHost(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "valid domain without scheme",
			input:       "foo.atlassian.net",
			expected:    "https://foo.atlassian.net",
			shouldError: false,
		},
		{
			name:        "valid domain with https scheme",
			input:       "https://foo.atlassian.net",
			expected:    "https://foo.atlassian.net",
			shouldError: false,
		},
		{
			name:        "valid domain with http scheme (should convert to https)",
			input:       "http://foo.atlassian.net",
			expected:    "https://foo.atlassian.net",
			shouldError: false,
		},
		{
			name:        "domain with path",
			input:       "foo.atlassian.net/some/path",
			expected:    "https://foo.atlassian.net/some/path",
			shouldError: false,
		},
		{
			name:        "domain with port",
			input:       "foo.atlassian.net:8080",
			expected:    "https://foo.atlassian.net:8080",
			shouldError: false,
		},
		{
			name:        "full URL with port and path",
			input:       "https://foo.atlassian.net:8080/path",
			expected:    "https://foo.atlassian.net:8080/path",
			shouldError: false,
		},
		{
			name:          "empty string",
			input:         "",
			shouldError:   true,
			errorContains: "host cannot be empty",
		},
		{
			name:        "localhost",
			input:       "localhost",
			expected:    "https://localhost",
			shouldError: false,
		},
		{
			name:        "localhost with port",
			input:       "localhost:8080",
			expected:    "https://localhost:8080",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeAndValidateHost(tt.input)

			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
