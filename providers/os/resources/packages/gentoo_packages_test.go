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

func TestParseGentooPackages(t *testing.T) {
	f, err := os.Open("testdata/gentoo_qlist.txt")
	require.NoError(t, err)
	defer f.Close()

	m, err := ParseGentooPackages(nil, f)
	require.NoError(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of packages")

	p := m[10] // net-misc/curl:8.4.0
	assert.Equal(t, "net-misc/curl", p.Name)
	assert.Equal(t, "8.4.0", p.Version)
	assert.Equal(t, "gentoo", p.Format)
	assert.Contains(t, p.PUrl, "pkg:ebuild/net-misc/curl@8.4.0")
}

func TestParseGentooPackagesRevision(t *testing.T) {
	f, err := os.Open("testdata/gentoo_qlist.txt")
	require.NoError(t, err)
	defer f.Close()

	m, err := ParseGentooPackages(nil, f)
	require.NoError(t, err)

	// net-misc/dhcpcd:10.0.5-r1
	p := m[11]
	assert.Equal(t, "net-misc/dhcpcd", p.Name)
	assert.Equal(t, "10.0.5-r1", p.Version)
	assert.Contains(t, p.PUrl, "pkg:ebuild/net-misc/dhcpcd@10.0.5-r1")
}

func TestParsePortageDB(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	pkgs, err := ParsePortageDB(nil, afs, "testdata/portage")
	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	// Sort order depends on OS readdir — find by name
	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	curl := byName["net-misc/curl"]
	assert.Equal(t, "8.4.0", curl.Version)
	assert.Equal(t, "gentoo", curl.Format)
	assert.Contains(t, curl.PUrl, "pkg:ebuild/net-misc/curl@8.4.0")
	assert.Contains(t, curl.Description, "Client and Server")

	dhcpcd := byName["net-misc/dhcpcd"]
	assert.Equal(t, "10.0.5-r1", dhcpcd.Version)
	assert.Contains(t, dhcpcd.Description, "DHCP client")

	audio := byName["acct-group/audio"]
	assert.Equal(t, "0-r2", audio.Version)
}

func TestSplitPortageDirName(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
	}{
		{"curl-8.4.0", "curl", "8.4.0"},
		{"dhcpcd-10.0.5-r1", "dhcpcd", "10.0.5-r1"},
		{"audio-0-r2", "audio", "0-r2"},
		{"libmnl-1.0.5", "libmnl", "1.0.5"},
		{"nghttp2-1.57.0", "nghttp2", "1.57.0"},
		{"iputils-20221126-r1", "iputils", "20221126-r1"},
	}
	for _, tt := range tests {
		name, version := splitPortageDirName(tt.input)
		assert.Equal(t, tt.name, name, "splitPortageDirName(%q) name", tt.input)
		assert.Equal(t, tt.version, version, "splitPortageDirName(%q) version", tt.input)
	}
}

func TestSplitCategoryName(t *testing.T) {
	cat, name := splitCategoryName("net-misc/curl")
	assert.Equal(t, "net-misc", cat)
	assert.Equal(t, "curl", name)

	cat, name = splitCategoryName("acct-group/audio")
	assert.Equal(t, "acct-group", cat)
	assert.Equal(t, "audio", name)

	cat, name = splitCategoryName("nocat")
	assert.Equal(t, "", cat)
	assert.Equal(t, "nocat", name)
}
