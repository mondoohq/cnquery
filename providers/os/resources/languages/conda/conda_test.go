// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conda

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCondaMeta(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	pkgs, fps := ParseCondaMeta(afs, "testdata/conda-meta")
	require.Len(t, pkgs, 3)
	require.NotEmpty(t, fps)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "1.26.4", byName["numpy"])
	assert.Equal(t, "2.2.1", byName["pandas"])
	assert.Equal(t, "3.12.2", byName["python"])

	// Check PURL with channel normalization
	for _, p := range pkgs {
		if p.Name == "numpy" {
			assert.Equal(t, "pkg:conda/pkgs/main/numpy@1.26.4", p.Purl)
		}
	}
}

func TestParseEnvironmentYml(t *testing.T) {
	f, err := os.Open("testdata/environment.yml")
	require.NoError(t, err)
	defer f.Close()

	pkgs, err := ParseEnvironmentYml(f, "testdata/environment.yml")
	require.NoError(t, err)
	// Should have 4 conda deps (pip sub-section is skipped)
	require.Len(t, pkgs, 4)

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	assert.Equal(t, "1.26.4", byName["numpy"])
	assert.Equal(t, "2.2.1", byName["pandas"])
	assert.Equal(t, "1.4.0", byName["scikit-learn"])
	assert.Equal(t, "3.12.2", byName["python"])
}

func TestParseCondaDep(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"numpy=1.26.4=py312h5b0bcb5_0", "numpy", "1.26.4"},
		{"pandas=2.2.1", "pandas", "2.2.1"},
		{"python", "python", ""},
		{"scikit-learn=1.4.0", "scikit-learn", "1.4.0"},
	}
	for _, tt := range tests {
		name, version := parseCondaDep(tt.input)
		assert.Equal(t, tt.name, name, "parseCondaDep(%q) name", tt.input)
		assert.Equal(t, tt.version, version, "parseCondaDep(%q) version", tt.input)
	}
}

func TestNormalizeChannel(t *testing.T) {
	assert.Equal(t, "pkgs/main", normalizeChannel("https://repo.anaconda.com/pkgs/main"))
	assert.Equal(t, "conda-forge", normalizeChannel("https://conda.anaconda.org/conda-forge"))
	assert.Equal(t, "", normalizeChannel(""))
	assert.Equal(t, "custom-channel", normalizeChannel("custom-channel"))
}

func TestNewPackageUrl(t *testing.T) {
	assert.Equal(t, "pkg:conda/pkgs/main/numpy@1.26.4", NewPackageUrl("numpy", "1.26.4", "https://repo.anaconda.com/pkgs/main"))
	assert.Equal(t, "pkg:conda/numpy@1.26.4", NewPackageUrl("numpy", "1.26.4", ""))
	assert.Equal(t, "", NewPackageUrl("", "1.0", ""))
}
