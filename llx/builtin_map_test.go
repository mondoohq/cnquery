// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFuzzyMapKey(t *testing.T) {
	m := map[string]any{
		"hello":        "world",
		"string-array": []any{"a", "b"},
		"dict":         map[string]any{"ee": 1, "ej": 2},
	}

	tests := []struct {
		name       string
		key        string
		wantMatch  string
		wantEmpty  bool
	}{
		// Single character typo - should find
		{"hallo->hello", "hallo", "hello", false},
		{"hell->hello", "hell", "hello", false},
		{"helloo->hello", "helloo", "hello", false},
		{"hillo->hello", "hillo", "hello", false},
		{"hlelo->hello", "hlelo", "hello", false},

		// Longer key with typo
		{"string-aray->string-array", "string-aray", "string-array", false},

		// Too different - should NOT match
		{"xyz->nothing", "xyz", "", true},
		{"completelyWrong->nothing", "completelyWrong", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFuzzyMapKey(tt.key, m)
			if tt.wantEmpty {
				assert.Empty(t, got, "expected no match for key %q", tt.key)
			} else {
				assert.Equal(t, tt.wantMatch, got, "expected %q for key %q", tt.wantMatch, tt.key)
			}
		})
	}
}
