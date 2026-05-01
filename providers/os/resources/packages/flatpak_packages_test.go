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

func TestParseFlatpakList(t *testing.T) {
	f, err := os.Open("testdata/flatpak_list.txt")
	require.NoError(t, err)
	defer f.Close()

	pkgs, err := ParseFlatpakList(f)
	require.NoError(t, err)
	require.Len(t, pkgs, 5)

	spotify := pkgs[0]
	assert.Equal(t, "com.spotify.Client", spotify.Name)
	assert.Equal(t, "1.2.31.564", spotify.Version)
	assert.Equal(t, "flathub", spotify.Origin)
	assert.Equal(t, "x86_64", spotify.Arch)
	assert.Equal(t, "flatpak", spotify.Format)
	assert.Contains(t, spotify.PUrl, "pkg:flatpak/flathub/com.spotify.Client@1.2.31.564")

	firefox := pkgs[1]
	assert.Equal(t, "org.mozilla.firefox", firefox.Name)
	assert.Equal(t, "131.0", firefox.Version)
}

func TestParseFlatpakDir(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	pkgs, err := parseFlatpakDir(afs, "testdata/flatpak/app")
	require.NoError(t, err)
	require.Len(t, pkgs, 2)

	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	spotify := byName["com.spotify.Client"]
	assert.Equal(t, "1.2.31.564", spotify.Version)
	assert.Equal(t, "x86_64", spotify.Arch)
	assert.Equal(t, "flathub", spotify.Origin)
	assert.Equal(t, "flatpak", spotify.Format)
	assert.Contains(t, spotify.PUrl, "pkg:flatpak/flathub/com.spotify.Client@1.2.31.564")

	firefox := byName["org.mozilla.firefox"]
	assert.Equal(t, "131.0", firefox.Version)
	assert.Equal(t, "flathub", firefox.Origin)
}

func TestNewFlatpakPurl(t *testing.T) {
	p := newFlatpakPurl("org.mozilla.firefox", "131.0", "flathub")
	assert.Equal(t, "pkg:flatpak/flathub/org.mozilla.firefox@131.0", p)

	p = newFlatpakPurl("com.spotify.Client", "1.2.31.564", "")
	assert.Equal(t, "pkg:flatpak/com.spotify.Client@1.2.31.564", p)

	p = newFlatpakPurl("", "1.0", "flathub")
	assert.Equal(t, "", p)
}
