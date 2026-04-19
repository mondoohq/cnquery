// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNixJSON(t *testing.T) {
	f, err := os.Open("testdata/nix_env.json")
	require.NoError(t, err)
	defer f.Close()

	pkgs, err := ParseNixJSON(f)
	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	curl := byName["curl"]
	assert.Equal(t, "8.7.1", curl.Version)
	assert.Equal(t, "nix", curl.Format)
	assert.Equal(t, "pkg:nix/curl@8.7.1", curl.PUrl)

	git := byName["git"]
	assert.Equal(t, "2.44.0", git.Version)
	assert.Equal(t, "pkg:nix/git@2.44.0", git.PUrl)
}

func TestParseNixStore(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	pkgs, err := ParseNixStore(afs, "testdata/nix_store")
	require.NoError(t, err)
	require.Len(t, pkgs, 4)

	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	curl := byName["curl"]
	assert.Equal(t, "8.7.1", curl.Version)
	assert.Equal(t, "nix", curl.Format)
	assert.Equal(t, "pkg:nix/curl@8.7.1", curl.PUrl)

	// Compound name with version
	requests := byName["python3.11-requests"]
	assert.Equal(t, "2.31.0", requests.Version)
}

func TestSplitNixNameVersion(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"curl-8.7.1", "curl", "8.7.1"},
		{"python3.11-requests-2.31.0", "python3.11-requests", "2.31.0"},
		{"git-2.44.0", "git", "2.44.0"},
		{"openssl-3.3.0", "openssl", "3.3.0"},
		{"nix-2.18.1", "nix", "2.18.1"},
		{"noversion", "noversion", ""},
	}
	for _, tt := range tests {
		name, version := splitNixNameVersion(tt.input)
		assert.Equal(t, tt.name, name, "splitNixNameVersion(%q) name", tt.input)
		assert.Equal(t, tt.version, version, "splitNixNameVersion(%q) version", tt.input)
	}
}

func TestNewNixPurl(t *testing.T) {
	assert.Equal(t, "pkg:nix/curl@8.7.1", newNixPurl("curl", "8.7.1"))
	assert.Equal(t, "pkg:nix/git@2.44.0", newNixPurl("git", "2.44.0"))
	assert.Equal(t, "", newNixPurl("", "1.0"))
}
