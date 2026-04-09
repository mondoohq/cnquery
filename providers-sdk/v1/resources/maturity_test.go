// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaturityLevel(t *testing.T) {
	tests := []struct {
		input string
		level int
	}{
		{"", 0},
		{"stable", 0},
		{"experimental", 1},
		{"preview", 2},
		{"deprecated", 3},
		{"eol", 4},
		{"bogus", -1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.level, MaturityLevel(tt.input))
		})
	}
}

func TestEffectiveMaturity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected string
	}{
		{"both empty", "", "", ""},
		{"both stable", "stable", "stable", ""},
		{"empty and stable", "", "stable", ""},
		{"experimental wins over empty", "experimental", "", "experimental"},
		{"empty and deprecated", "", "deprecated", "deprecated"},
		{"experimental and deprecated", "experimental", "deprecated", "deprecated"},
		{"eol and preview", "eol", "preview", "eol"},
		{"preview and experimental", "preview", "experimental", "preview"},
		{"deprecated and eol", "deprecated", "eol", "eol"},
		{"same non-stable", "preview", "preview", "preview"},
		{"stable string and experimental", "stable", "experimental", "experimental"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, EffectiveMaturity(tt.a, tt.b))
		})
	}
}

func TestEffectiveFieldMaturity(t *testing.T) {
	t.Run("both stable", func(t *testing.T) {
		ri := &ResourceInfo{}
		f := &Field{}
		assert.Equal(t, "", EffectiveFieldMaturity(ri, f))
	})

	t.Run("resource experimental", func(t *testing.T) {
		ri := &ResourceInfo{Maturity: "experimental"}
		f := &Field{}
		assert.Equal(t, "experimental", EffectiveFieldMaturity(ri, f))
	})

	t.Run("field deprecated", func(t *testing.T) {
		ri := &ResourceInfo{}
		f := &Field{Maturity: "deprecated"}
		assert.Equal(t, "deprecated", EffectiveFieldMaturity(ri, f))
	})

	t.Run("resource experimental field deprecated", func(t *testing.T) {
		ri := &ResourceInfo{Maturity: "experimental"}
		f := &Field{Maturity: "deprecated"}
		assert.Equal(t, "deprecated", EffectiveFieldMaturity(ri, f))
	})
}

func TestValidateMaturity(t *testing.T) {
	valid := []string{"", "experimental", "preview", "stable", "deprecated", "eol"}
	for _, m := range valid {
		t.Run("valid_"+m, func(t *testing.T) {
			require.NoError(t, ValidateMaturity(m))
		})
	}

	invalid := []string{"bogus", "Experimental", "STABLE", "unknown"}
	for _, m := range invalid {
		t.Run("invalid_"+m, func(t *testing.T) {
			require.Error(t, ValidateMaturity(m))
		})
	}
}

func TestMaturityLabel(t *testing.T) {
	assert.Equal(t, "", MaturityLabel(""))
	assert.Equal(t, "", MaturityLabel("stable"))
	assert.Equal(t, "Experimental", MaturityLabel("experimental"))
	assert.Equal(t, "Preview", MaturityLabel("preview"))
	assert.Equal(t, "Deprecated", MaturityLabel("deprecated"))
	assert.Equal(t, "EOL", MaturityLabel("eol"))
}
